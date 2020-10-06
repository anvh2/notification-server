package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/anvh2/notification-server/middlewares"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AuthenticationFunc ...
type AuthenticationFunc func(r *http.Request) (string, error)

// Message ...
type Message struct {
	Topic string `json:"topic,omitempty"`
	Data  string `json:"data,omitempty"`
}

// String ...
func (m *Message) String() string {
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

type notify struct {
	channelID string
	message   Message
}

type client struct {
	channelID   string
	connected   bool
	messageChan chan Message
}

// Broker holds open client  connections,
// listen for incoming events on it Notifier channel
// and broadcast event data to all registered connections
type Broker struct {
	logger          *zap.Logger
	port            int
	ccu             int
	mux             sync.Mutex
	authFunc        AuthenticationFunc
	notifier        chan notify
	newClientChan   chan client
	closeClientChan chan string
	clients         map[string]client
	quitc           chan struct{}
}

// NewBroker ...
func NewBroker(logger *zap.Logger, port int, authFunc AuthenticationFunc) *Broker {
	return &Broker{
		logger:          logger,
		port:            port,
		ccu:             0,
		mux:             sync.Mutex{},
		authFunc:        authFunc,
		notifier:        make(chan notify, 1000),
		newClientChan:   make(chan client),
		closeClientChan: make(chan string),
		clients:         make(map[string]client),
		quitc:           make(chan struct{}),
	}
}

// Run -
func (b *Broker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("[Run] recover sse server", zap.Any("r", r))
		}
	}()

	go func() {
		for {
			select {
			case client := <-b.newClientChan: // TODO: must be check connection is already exist and remove or ignore
				b.clients[client.channelID] = client
				b.logger.Info("client registered", zap.String("channleID", client.channelID))

			case clientID := <-b.closeClientChan:
				delete(b.clients, clientID)
				b.logger.Info("client deregistered", zap.String("channelID", clientID))

			case notify := <-b.notifier:
				for _, client := range b.clients {
					if notify.channelID == client.channelID {
						client.messageChan <- notify.message
					}
				}

			case <-b.quitc:
				return
			}
		}
	}()

	b.logger.Info("[Run] Start sse server", zap.Int("port", b.port), zap.String("version", viper.GetString("service_version")))

	mux := http.NewServeMux()
	mux.Handle("/", middlewares.RecoverHTTP(b))

	return http.ListenAndServe(fmt.Sprintf(":%d", b.port), mux)
}

// ServeHTTP ...
func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	dump, _ := httputil.DumpRequest(req, true)
	b.logger.Info("Request", zap.String("req", string(dump)))

	// support flusing
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusNotImplemented)
		return
	}

	clientID, err := b.authFunc(req)
	if err != nil {
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		b.logger.Error("[ServeHTTP] failed to authenticated connection", zap.Error(err))
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no")

	origin := req.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	rw.Header().Set("Access-Control-Allow-Origin", origin)
	rw.Header().Set("Access-Control-Allow-Credentials", "true")
	rw.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS, POST, PUT")
	rw.Header().Set("Access-Control-Allow-Headers", "Cache-Control, Access-Control-Allow-Headers, Origin, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Allow-Methods, Access-Control-Allow-Origin")

	rw.WriteHeader(http.StatusOK)
	flusher.Flush()

	// register its own message channel
	client := client{
		channelID:   clientID,
		connected:   true,
		messageChan: make(chan Message),
	}

	b.newClientChan <- client

	go b.incrConn()

	defer func() {
		b.closeClientChan <- client.channelID
	}()

	// listen to connection close
	notify := rw.(http.CloseNotifier).CloseNotify()

	go func() {
		select {
		case <-notify:
			b.closeClientChan <- client.channelID
		case <-b.quitc:
			return
		}
	}()

	b.logger.Info("[ServeHTTP] waiting for messages broadcast", zap.String("clientID", clientID))
	for {
		message := <-client.messageChan
		res, err := json.Marshal(message)
		if err != nil {
			http.Error(rw, "Unfortunate!", http.StatusInternalServerError)
		}

		b.logger.Info("push message", zap.String("message", string(res)))

		rw.Write([]byte("event: message\n"))
		rw.Write([]byte(fmt.Sprintf("data: %s\n\n", string(res))))

		flusher.Flush()
	}
}

func (b *Broker) incrConn() {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.ccu++
}

// PushMessage ...
func (b *Broker) PushMessage(userID, event, data string) {
	b.notifier <- notify{
		channelID: userID,
		message: Message{
			Topic: event,
			Data:  data,
		},
	}
}

// Close -
func (b *Broker) Close() {
	close(b.quitc)
}
