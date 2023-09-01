package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	master, err := utils.ReadStringEnvVar("MASTER")
	utils.FailOnError("Failed to read environment variables", err)
	rabbitHost, err := utils.ReadStringEnvVar("RABBIT_HOST")
	utils.FailOnError("Failed to read environment variables", err)
	rabbitUser := utils.ReadStringEnvVarOr("RABBIT_USER", "guest")
	rabbitPass := utils.ReadStringEnvVarOr("RABBIT_PASSWORD", "guest")
	nodePort, err := utils.ReadIntEnvVar("NODE_PORT")
	utils.FailOnError("Failed to read environment variables", err)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodePort))
	utils.FailOnError("Failed to listen for node server", err)

	// Connect to RabbitMQ
	queue := fmt.Sprintf("amqp://%s:%s@%s:5672/", rabbitUser, rabbitPass, rabbitHost)
	queueConn, err := amqp.Dial(queue)
	utils.FailOnError("Could not connect to RabbitMQ", err)
	defer queueConn.Close()
	ch, err := queueConn.Channel()
	utils.FailOnError("Failed to open a channel to RabbitMQ", err)
	defer ch.Close()

	n := pkg.Node{
		State: &proto.State{
			Phase: int32(pkg.Wait),
			Data:  make(map[int32]float64),
		},
		Role: pkg.Master,
		Queue: pkg.Queue{
			Conn:    queueConn,
			Channel: ch,
		},
	}
	workQueueName := "work"
	resultQueueName := "result"

	// Contact master node to join the network
	masterClient, err := utils.NodeCall(master)
	defer masterClient.CancelFunc()
	defer masterClient.Conn.Close()
	utils.FailOnError("Could not create connection to the masterClient node", err)
	join, err := masterClient.Client.NodeJoin(
		masterClient.Ctx,
		&wrapperspb.StringValue{Value: lis.Addr().String()},
	)
	if err != nil {
		// There is no node at the address -> creating a new network
		// This node will be the master
		log.Printf("No master node found at %s\n", master)
	} else {
		// Ther is a master node -> this node will be a worker
		n.Role = pkg.Worker
		n.Master = master
		n.State = join.State
		workQueueName = join.WorkQueue
		resultQueueName = join.ResultQueue
	}
	work, err := utils.DeclareQueue(workQueueName, ch)
	utils.FailOnError("Failed to declare 'work' queue", err)
	n.Queue.Work = &work
	result, err := utils.DeclareQueue(resultQueueName, ch)
	utils.FailOnError("Failed to declare 'result' queue", err)
	n.Queue.Result = &result

	// Running gRPC server for internal network communication in a goroutine
	go func() {
		// Creating gRPC server
		server := grpc.NewServer()
		proto.RegisterNodeServer(server, &pkg.NodeServerImpl{Node: &n})
		log.Printf("Starting %s node at %v\n", pkg.RoleToString(n.Role), lis.Addr())
		err = server.Serve(lis)
		utils.FailOnError("Failed to serve", err)
	}()
	// Node Update
	n.Update()
}
