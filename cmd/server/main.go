package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// Default value
const HEALTH_CHECK_DEFAULT = 5000

func main() {
	port, master, queue, healthCheck, err := ReadEnvVars()
	pkg.FailOnError("Failed to read environment variables", err)

	// Connect to RabbitMQ
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

	// Running gRPC server for internal network communication in a goroutine
	go func() {
		// Creating gRPC server
		lis, err := net.Listen("tcp", fmt.Sprintf("%d", port))
		pkg.FailOnError("Failed to listen", err)
		server := grpc.NewServer()
		proto.RegisterNodeServer(server, &pkg.NodeServerImpl{Node: &n})
		log.Printf("Starting %s node at %s\n", pkg.RoleToString(n.Role), lis.Addr().String())
		err = server.Serve(lis)
		pkg.FailOnError("Failed to serve", err)
	}()
	// Node update (computation phase)
	go func() {
		if err = n.Update(); err != nil {
			log.Fatalf("Node update error: %v", err)
		}
	}()
	if n.Role == pkg.Master {
		// Running gRPC server for client communication in a goroutine
		lis, err := net.Listen("tcp", "0")
		pkg.FailOnError("Failed to listen", err)
		s := grpc.NewServer()
		proto.RegisterApiServer(s, &pkg.ApiServerImpl{Node: &n})
		log.Printf("Starting api server at %s\n", lis.Addr().String())
		err = s.Serve(lis)
		pkg.FailOnError("Failed to serve", err)
	}
	if n.Role == pkg.Worker {
		// Worker Health Check
		go func() {
			for {
				n.WorkerHealthCheck()
				time.Sleep(time.Duration(healthCheck) * time.Millisecond)
			}
		}()
	}
}

// Returns, in order:
// - port: where the node will start - 0 for automatic port assignment
// - master: expected address of the master node
// - queue: RabbitMQ connection string
// - healthCheck: how often a worker node contact the master node for health check
// - err: if anything goes wrong
func ReadEnvVars() (port int, master string, queue string, healthCheckMilli int, err error) {
	master = os.Getenv("MASTER")
	if master == "" {
		err = errors.New("MASTER not set")
		return
	}
	queue = os.Getenv("QUEUE")
	if queue == "" {
		err = errors.New("QUEUE not set")
		return
	}
	portString := os.Getenv("PORT")
	if portString == "" {
		portString = "0"
	}
	port, err = strconv.Atoi(portString)
	if err != nil {
		return
	}
	healthCheckMilliString := os.Getenv("HEALTH_CHECK")
	if healthCheckMilliString == "" {
		healthCheckMilli = HEALTH_CHECK_DEFAULT
		return
	}
	healthCheckMilli, err = strconv.Atoi(healthCheckMilliString)
	if err != nil {
		healthCheckMilli = HEALTH_CHECK_DEFAULT
		log.Printf("Could not convert %s in an integer. Using default value of %d\n", healthCheckMilliString, HEALTH_CHECK_DEFAULT)
	}
	return
}
