package utils

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client[T interface{}] struct {
	Client T
	Ctx    context.Context
	conn   *grpc.ClientConn
	cancel context.CancelFunc
}

// Utility function to create a gRPC client to `url`
// Has to be closed (`c.Close()`)
func NodeCall(url string) (Client[proto.NodeClient], error) {
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return Client[proto.NodeClient]{}, err
	}
	client := proto.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return Client[proto.NodeClient]{
		conn:   conn,
		Client: client,
		Ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Utility function to create a gRPC client to `url`
// Has to be closed (`c.Close()`)
func ApiCall(url string) (Client[proto.APIClient], error) {
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return Client[proto.APIClient]{}, err
	}
	client := proto.NewAPIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return Client[proto.APIClient]{
		conn:   conn,
		Client: client,
		Ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c Client[T]) Close() {
	c.cancel()
	c.conn.Close()
}

func FailOnError(format string, err error, v ...any) {
	if err != nil {
		log.Fatalf("%s: %v", fmt.Sprintf(format, v...), err)
	}
}

func ReadStringFromStdin(question string) string {
	var input string
	fmt.Print(question)
	fmt.Scanln(&input)
	return input
}

func ReadFloat64FromStdin(question string) float64 {
	for {
		input := ReadStringFromStdin(question)
		value, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("Input was not a number. Try again")
			continue
		}
		return value
	}
}
