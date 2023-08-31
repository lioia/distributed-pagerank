package main

import (
	"fmt"
	"log"
	"net"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	master, err := pkg.ReadStringEnvVar("MASTER")
	pkg.FailOnError("Failed to read environment variables", err)
	rabbitHost, err := pkg.ReadStringEnvVar("RABBIT_HOST")
	pkg.FailOnError("Failed to read environment variables", err)
	rabbitUser := pkg.ReadStringEnvVarOr("RABBIT_USER", "guest")
	rabbitPass := pkg.ReadStringEnvVarOr("RABBIT_PASSWORD", "guest")
	nodePort, err := pkg.ReadIntEnvVar("NODE_PORT")
	pkg.FailOnError("Failed to read environment variables", err)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodePort))
	pkg.FailOnError("Failed to listen for node server", err)

	// Connect to RabbitMQ
	queue := fmt.Sprintf("amqp://%s:%s@%s:5672/", rabbitUser, rabbitPass, rabbitHost)
	queueConn, err := amqp.Dial(queue)
	pkg.FailOnError("Could not connect to RabbitMQ", err)
	defer queueConn.Close()
	ch, err := queueConn.Channel()
	pkg.FailOnError("Failed to open a channel to RabbitMQ", err)
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
	masterClient, err := pkg.NodeCall(master)
	defer masterClient.CancelFunc()
	defer masterClient.Conn.Close()
	pkg.FailOnError("Could not create connection to the masterClient node", err)
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
		n.C = join.C
		n.Threshold = join.Threshold
		workQueueName = join.WorkQueue
		resultQueueName = join.ResultQueue
	}
	work, err := pkg.DeclareQueue(workQueueName, ch)
	pkg.FailOnError("Failed to declare 'work' queue", err)
	n.Queue.Work = &work
	result, err := pkg.DeclareQueue(resultQueueName, ch)
	pkg.FailOnError("Failed to declare 'result' queue", err)
	n.Queue.Result = &result

	// Running gRPC server for internal network communication in a goroutine
	go func() {
		// Creating gRPC server
		server := grpc.NewServer()
		proto.RegisterNodeServer(server, &pkg.NodeServerImpl{Node: &n})
		log.Printf("Starting %s node at %v\n", pkg.RoleToString(n.Role), lis.Addr())
		err = server.Serve(lis)
		pkg.FailOnError("Failed to serve", err)
	}()
	// Running gRPC server for client communication in a goroutine
	if n.Role == pkg.Master {
		apiPort, err := pkg.ReadIntEnvVar("API_PORT")
		pkg.FailOnError("Failed to read environment variables", err)
		go func() {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", apiPort))
			pkg.FailOnError("Failed to listen for API server", err)
			s := grpc.NewServer()
			proto.RegisterApiServer(s, &pkg.ApiServerImpl{Node: &n})
			log.Printf("Starting API server at %v\n", lis.Addr())
			err = s.Serve(lis)
			pkg.FailOnError("Failed to serve", err)
		}()
	}
	// Node Update
	n.Update()
}
