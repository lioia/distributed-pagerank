package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Client proto.NodeClient
	Ctx    context.Context
	conn   *grpc.ClientConn
	cancel context.CancelFunc
}

// Utility function to create a gRPC client to `url`
// Has to be closed (`c.Close()`)
func NodeCall(url string) (Client, error) {
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return Client{}, err
	}
	client := proto.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return Client{
		conn:   conn,
		Client: client,
		Ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c Client) Close() {
	c.cancel()
	c.conn.Close()
}

func FailOnError(msg string, err error) {
	if err != nil {
		log.Panicf("%s: %v", msg, err)
	}
}

func ReadStringEnvVar(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s not set", name)
	}
	return value, nil
}

func ReadIntEnvVar(name string) (int, error) {
	valueStr, err := ReadStringEnvVar(name)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("Could not convert %s to a number: %v", name, err)
	}
	return value, nil
}

func ReadStringEnvVarOr(name string, or string) string {
	value, err := ReadStringEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func ReadIntEnvVarOr(name string, or int) int {
	value, err := ReadIntEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func ReadFloat64FromStdin(question string) float64 {
	var input string
	for {
		fmt.Print(question)
		fmt.Scanln(&input)
		value, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("Input was not a number. Try again")
			continue
		}
		return value
	}
}
