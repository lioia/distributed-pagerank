package pkg

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"
	amqp "github.com/rabbitmq/amqp091-go"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

func DeclareQueue(name string, ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

func FailOnNack(d amqp.Delivery, err error) {
	fmt.Printf("Could not marshal result: %v", err)
	// Message will be re-added to the queue
	if err = d.Nack(false, true); err != nil {
		log.Fatalf("Could not NACK to message queue: %v", err)
	}
}

func EmptyQueue(ch *amqp.Channel, name string) {
	msgs, err := ch.Consume(
		name,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	FailOnError("Could not register a consumer", err)

	for msg := range msgs {
		// Acknowledge the message to remove it from the queue
		err := msg.Ack(false)
		if err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
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
		pkg.FailOnError("Coult not read from stdin", err)
	}
	return value
}
