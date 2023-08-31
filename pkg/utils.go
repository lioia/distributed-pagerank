package pkg

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client[T interface{}] struct {
	Conn       *grpc.ClientConn
	Client     T
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

// User has to `defer CancelFunc()` and `defer Conn.Close()`
func NodeCall(url string) (Client[proto.NodeClient], error) {
	var clientInfo Client[proto.NodeClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := proto.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	clientInfo.Conn = conn
	clientInfo.Client = client
	clientInfo.Ctx = ctx
	clientInfo.CancelFunc = cancel
	return clientInfo, nil
}

// User has to `defer CancelFunc()` and `defer Conn.Close()`
func ApiCall(url string) (Client[proto.ApiClient], error) {
	var clientInfo Client[proto.ApiClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := proto.NewApiClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	clientInfo.Conn = conn
	clientInfo.Client = client
	clientInfo.Ctx = ctx
	clientInfo.CancelFunc = cancel
	return clientInfo, nil
}

func FailOnError(msg string, err error) {
	if err != nil {
		log.Panicf("%s: %v", msg, err)
	}
}

func ReadFromStdin(question string) (string, error) {
	fmt.Print(question)
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

func ReadFromStdinAndFail(question string) string {
	value, err := ReadFromStdin(question)
	if err != nil {
		FailOnError("Coult not read from stdin", err)
	}
	return strings.TrimRight(value, "\n")
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
