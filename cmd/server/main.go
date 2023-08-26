package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

var port int          // Port where the node will start
var master string     // Expected connection string of the master node
var c float64         // PageRank `c` parameter
var threshold float64 // PageRank threshold
var queue string      // Queue connection string

func init() {
	flag.IntVar(&port, "port", 0, "Port") // 0: automatic port assignment
	flag.StringVar(&master, "master", "127.0.0.1:1234", "Master Connection")
	flag.Float64Var(&c, "c", 0.85, "c variable")
	flag.Float64Var(&threshold, "threshold", 0.0001, "Threshold")
	flag.StringVar(&queue, "queue", "amqp://guest:guest@localhost:5672", "Queue Connection String")
}

func main() {
	flag.Parse()

	// TODO: read configuration file

	// Connect to RabbitMQ
	queueConn, err := amqp.Dial(queue)
	pkg.FailOnError("Could not connect to RabbitMQ", err)
	defer queueConn.Close()
	ch, err := queueConn.Channel()
	pkg.FailOnError("Failed to open a channel to RabbitMQ", err)
	defer ch.Close()

	n := pkg.Node{
		State:     &proto.State{Phase: int32(pkg.Wait)},
		Role:      pkg.Master,
		C:         0.85,  // TODO: configurable variable
		Threshold: 0.001, // TODO: configurable variable
		Data:      make(map[int32]float64),
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
	join, err := masterClient.Client.NodeJoin(masterClient.Ctx, nil)
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

	// Creating gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf("%d", port))
	pkg.FailOnError("Failed to listen", err)
	server := grpc.NewServer()
	proto.RegisterNodeServer(server, &pkg.NodeServerImpl{Node: &n})
	log.Printf("Starting %s node at %s\n", pkg.RoleToString(n.Role), lis.Addr().String())
	// Running gRPC server in a goroutine
	go func() {
		err = server.Serve(lis)
		pkg.FailOnError("Failed to serve", err)
	}()
	// Node update (computation phase)
	go func() {
		if err = n.Update(); err != nil {
			log.Fatalf("Node update error: %v", err)
		}
	}()
	if n.Role == pkg.Worker {
		// Worker Health Check
		go func() {
			for {
				n.WorkerHealthCheck()
				time.Sleep(5 * time.Second) // TODO: configurable parameters
			}
		}()
	}
}
