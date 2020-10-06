package integration

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	pb "github.com/anvh2/notification-server/grpc-gen"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

var (
	ctx    = context.Background()
	client = pb.NewNotificationServiceClient(newConn())
)

func newConn() *grpc.ClientConn {
	conn, err := grpc.Dial(":55100", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return conn
}

func TestPushMessage(t *testing.T) {
	res, err := client.PushMessage(ctx, &pb.PushMessageRequest{
		UserID: "123",
		Event:  "event_test",
		Data:   "Hello world",
	})
	assert.Nil(t, err)
	fmt.Println(res)
}

func BenchmarkConnections(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cmd := exec.Command("curl", "http://localhost:55102?token=123")
			cmd.Run()
		}
	})
}
