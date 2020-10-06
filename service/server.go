package backend

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/anvh2/notification-server/grpc-gen"
	"github.com/anvh2/notification-server/sse"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Server ...
type Server struct {
	logger *zap.Logger
	broker *sse.Broker
}

// NewServer ...
func NewServer() *Server {
	logger, err := newLogger(viper.GetString("server.log_path"))
	if err != nil {
		log.Fatal("failed to new logger\n", err)
	}

	server := &Server{
		logger: logger,
	}

	server.broker = sse.NewBroker(logger, viper.GetInt("server.http_port"), server.authen)

	return server
}

// Run ...
func (s *Server) Run() error {
	port := viper.GetInt("server.grpc_port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.logger.Fatal("failed to listen tcp", zap.Error(err))
	}

	// TODO: authen later
	opts := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_recovery.UnaryServerInterceptor(),
	))

	server := grpc.NewServer(opts)
	pb.RegisterNotificationServiceServer(server, s)

	go func() {
		if err := server.Serve(lis); err != nil {
			s.logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	go func() {
		if err := s.broker.Run(); err != nil {
			s.logger.Fatal("failed to start sse server", zap.Error(err))
		}
	}()

	s.logger.Info("start listen", zap.Int("port", port))

	sig := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Println("Shuting down...")
		s.broker.Close()
		close(done)
	}()

	fmt.Println("Server is listening\nCtr-c to interup...")
	<-done
	fmt.Println("Shutdown")
	return nil
}

// PushMessage ...
func (s *Server) PushMessage(ctx context.Context, req *pb.PushMessageRequest) (*pb.PushMessageResponse, error) {
	if req.UserID == "" {
		s.logger.Error("[PushMessage] empty user id")
		return &pb.PushMessageResponse{
			Code:    -1,
			Message: "EMPTY_USER_ID",
		}, nil
	} else if req.Event == "" {
		s.logger.Error("[PushMessage] emtpy event name", zap.String("userID", req.UserID))
		return &pb.PushMessageResponse{
			Code:    -1,
			Message: "EMPTY_EVENT",
		}, nil
	} else if req.Data == "" {
		s.logger.Error("[PushMessage] emtpy data", zap.String("userID", req.UserID))
		return &pb.PushMessageResponse{
			Code:    -1,
			Message: "EMPTY_DATA",
		}, nil
	}

	s.logger.Info("[PushMessage] push message to client", zap.String("req", req.String()))

	s.broker.PushMessage(req.UserID, req.Event, req.Data)

	return &pb.PushMessageResponse{
		Code:    1,
		Message: "OK",
	}, nil
}

func newLogger(path string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	config.OutputPaths = []string{path}
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "message"
	config.DisableStacktrace = true

	return config.Build()
}

func newGrpcClient() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	optsRetry := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(50 * time.Millisecond)),
		grpc_retry.WithCodes(codes.Unavailable),
		grpc_retry.WithMax(3),
		grpc_retry.WithPerRetryTimeout(3 * time.Second),
	}

	opts = append(opts,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.FailFast(false)),
		grpc.WithChainUnaryInterceptor(grpc_retry.UnaryClientInterceptor(optsRetry...)),
	)

	return grpc.Dial(viper.GetString("server.internal_api_addr"), opts...)
}

func (s *Server) authen(req *http.Request) (string, error) {
	token := req.URL.Query().Get("token")

	// TODO: decrypt token here

	return token, nil
}
