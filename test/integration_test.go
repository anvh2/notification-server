package integration

import (
	"context"
	"fmt"
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
