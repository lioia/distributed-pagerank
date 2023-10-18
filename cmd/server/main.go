package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/lioia/distributed-pagerank/pkg/node"
	"github.com/lioia/distributed-pagerank/pkg/utils"
	"github.com/lioia/distributed-pagerank/proto"
	gonanoid "github.com/matoous/go-nanoid/v2"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	// Read environment variables
	env := utils.ReadEnvVars()
	utils.InitLog(env.NodeLog, env.ServerLog)

	// Create connection
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", env.Port))
	utils.FailOnError("Failed to listen for node server", err)
	// lis.Close in goroutine

	realPort := env.Port
	realHost := env.Host
	if env.Port == 0 {
		realPort = lis.Addr().(*net.TCPAddr).Port
	}
	if env.Host == "" {
		realHost = lis.Addr().(*net.TCPAddr).IP.String()
	}

	// Connect to RabbitMQ
	queue := fmt.Sprintf("amqp://%s:%s@%s:5672/", env.RabbitUser, env.RabbitPass, env.RabbitHost)
	queueConn, err := amqp.Dial(queue)
	utils.FailOnError("Could not connect to RabbitMQ", err)
	defer queueConn.Close()
	ch, err := queueConn.Channel()
	utils.FailOnError("Failed to open a channel to RabbitMQ", err)
	defer ch.Close()

	id, _ := gonanoid.New()
	// Base node values
	n := node.Node{
		Id: id,
		State: &proto.State{
			Others: make(map[string]string),
		},
		Data: sync.Map{},
		Role: node.Master,
		Queue: node.Queue{
			Conn:    queueConn,
			Channel: ch,
		},
		Connection: fmt.Sprintf("%s:%d", realHost, realPort),
		Phase:      node.Wait,
	}

	// Contact master node to join the network
	client, err := utils.NodeCall(env.Master)
	utils.FailOnError("Failed to create connection to the master node", err)
	defer client.Close()
	join, err := client.Client.NodeJoin(
		client.Ctx,
		&wrapperspb.StringValue{Value: fmt.Sprintf("%s:%d", realHost, realPort)},
	)
	if err != nil {
		// There is no node at the address -> creating a new network
		// This node will be the master
		utils.NodeLog("master", "No master node found at %s", env.Master)
	} else {
		utils.NodeLog("worker", "Found master at %s", env.Master)
		n.InitializeWorker(env.Master, join)
		env.WorkQueue = join.WorkQueue
		env.ResultQueue = join.ResultQueue
	}
	// Queue declaration
	work, err := utils.DeclareQueue(env.WorkQueue, ch)
	utils.FailOnError("Failed to declare 'work' queue", err)
	n.Queue.Work = &work
	result, err := utils.DeclareQueue(env.ResultQueue, ch)
	utils.FailOnError("Failed to declare 'result' queue", err)
	n.Queue.Result = &result

	// Running gRPC server for internal network communication in a goroutine
	status := make(chan bool)
	go func() {
		// Creating gRPC server
		defer lis.Close()
		server := grpc.NewServer()
		proto.RegisterNodeServer(server, &node.NodeServerImpl{Node: &n})
		fmt.Printf("Starting %s node at %s:%d\n",
			node.RoleToString(n.Role), realHost, realPort)
		status <- true
		err = server.Serve(lis)
		utils.FailOnError("Failed to serve", err)
	}()
	// Waiting for gRPC server to start
	<-status
	// Node Update
	n.Update()
}
