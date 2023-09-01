package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/lioia/distributed-pagerank/graph"
	"github.com/lioia/distributed-pagerank/node"
	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	// Read environment variables
	master, err := utils.ReadStringEnvVar("MASTER")
	utils.FailOnError("Failed to read environment variables", err)
	rabbitHost, err := utils.ReadStringEnvVar("RABBIT_HOST")
	utils.FailOnError("Failed to read environment variables", err)
	rabbitUser := utils.ReadStringEnvVarOr("RABBIT_USER", "guest")
	rabbitPass := utils.ReadStringEnvVarOr("RABBIT_PASSWORD", "guest")
	port, err := utils.ReadIntEnvVar("PORT")
	utils.FailOnError("Failed to read environment variables", err)

	// Create connection
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	utils.FailOnError("Failed to listen for node server", err)
	// lis.Close in goroutine

	// Connect to RabbitMQ
	queue := fmt.Sprintf("amqp://%s:%s@%s:5672/", rabbitUser, rabbitPass, rabbitHost)
	queueConn, err := amqp.Dial(queue)
	utils.FailOnError("Could not connect to RabbitMQ", err)
	defer queueConn.Close()
	ch, err := queueConn.Channel()
	utils.FailOnError("Failed to open a channel to RabbitMQ", err)
	defer ch.Close()

	// Base node values
	n := node.Node{
		State: &proto.State{
			Phase: int32(node.Wait),
			Data:  make(map[int32]float64),
		},
		Role: node.Master,
		Queue: node.Queue{
			Conn:    queueConn,
			Channel: ch,
		},
	}
	// Default value for master
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
		c, threshold, graph, err := loadConfiguration()
		if err != nil {
			// Configuration could not be loaded
			log.Println("Configuration will asked later")
		} else {
			// Configuration loaded correctly
			n.State.C = c
			n.State.Threshold = threshold
			n.State.Graph = graph
		}
	} else {
		// Ther is a master node -> this node will be a worker
		n.Role = node.Worker
		n.Master = master
		n.State = join.State
		workQueueName = join.WorkQueue
		resultQueueName = join.ResultQueue
	}
	// Queue declaration
	work, err := utils.DeclareQueue(workQueueName, ch)
	utils.FailOnError("Failed to declare 'work' queue", err)
	n.Queue.Work = &work
	result, err := utils.DeclareQueue(resultQueueName, ch)
	utils.FailOnError("Failed to declare 'result' queue", err)
	n.Queue.Result = &result

	// Running gRPC server for internal network communication in a goroutine
	go func() {
		// Creating gRPC server
		defer lis.Close()
		server := grpc.NewServer()
		proto.RegisterNodeServer(server, &node.NodeServerImpl{Node: &n})
		log.Printf("Starting %s node at %v\n", node.RoleToString(n.Role), lis.Addr())
		err = server.Serve(lis)
		utils.FailOnError("Failed to serve", err)
	}()
	// Node Update
	n.Update()
}

// Load config.json (C, Threshold and graph file)
func loadConfiguration() (c float64, threshold float64, g map[int32]*proto.GraphNode, err error) {
	// Try to open the config.json file
	_, err = os.Open("config.json")
	if err != nil {
		log.Printf("Configuration file does not exists: %v", err)
		return
	}
	// File exists -> load configuration
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		log.Printf("Could not read configuration file: %v", err)
		return
	}
	// Parse config.json into a Golang struct
	var config node.Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		log.Printf("Could not parse configuration file: %v", err)
		return
	}
	// Check if it's a network resource or a local one
	if strings.HasPrefix(config.Graph, "http") {
		// Loading file from network
		var resp *http.Response
		resp, err = http.Get(config.Graph)
		if err != nil {
			log.Printf("Could not load network file at %s: %v", config.Graph, err)
			return
		}
		defer resp.Body.Close()
		// Read response body
		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Could not load body from request: %v", err)
		}
	} else {
		// Loading file from local filesystem
		bytes, err = os.ReadFile(config.Graph)
		if err != nil {
			log.Printf("Could not read graph at %s: %v", config.Graph, err)
			return
		}
	}
	// Parse graph file into graph representation
	g, err = graph.LoadGraphFromBytes(bytes)
	if err != nil {
		log.Printf("Could not load graph from %s: %v", config.Graph, err)
		return
	}
	c = config.C
	threshold = config.Threshold

	return
}
