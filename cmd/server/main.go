package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/pkg/services"
	"google.golang.org/grpc"
)

var port int              // Port where a new node will be created
var masterAddress string  // Expected address of the master node of the network
var masterPort int        // Expected port of the master node
var dampingFactor float64 // PageRank `c` parameter
var threshold float64     // PageRank threshold

func init() {
	flag.IntVar(&port, "port", 1234, "Node Port number")
	flag.StringVar(&masterAddress, "masterAddress", "127.0.0.1", "Master Node Address")
	flag.IntVar(&masterPort, "masterPort", 1234, "Master Node Port number")
	flag.Float64Var(&dampingFactor, "dampingFactor", 0.85, "Damping Factor (c)")
	flag.Float64Var(&threshold, "threshold", 0.0001, "Threshold")
}

func main() {
	flag.Parse()

	// TODO: read configuration file

	// Get local address from network interfaces
	address, err := pkg.GetAddressFromInterfaces()
	if err != nil {
		log.Fatalf("unable to determine local address: %v\n", err)
	}
	conn := services.ConnectionInfo{
		Address: address,
		Port:    int32(port),
	}

	node := Node{
		Phase:         Wait,
		Role:          Master,
		DampingFactor: 0.85, // TODO: configurable variable
		Connection:    &conn,
	}

	// Contact master node to join the network
	masterConn := services.ConnectionInfo{
		Address: masterAddress,
		Port:    int32(masterPort),
	}
	master, err := pkg.NodeCall(pkg.Url(&masterConn))
	defer master.CancelFunc()
	defer master.Conn.Close()
	if err != nil {
		log.Fatalf("could not create connection to the master: %v", err)
	}
	state, err := master.Client.HealthCheck(master.Ctx, nil)
	if err != nil {
		// There is no node at the address -> creating a new network
		// This node will be the master
		fmt.Printf("No master node found at %s\n", pkg.Url(&masterConn))
		fmt.Printf("Starting master node at %s\n", pkg.Url(&conn))
	} else {
		// Ther is a master node -> this node will be a worker
		node.Role = Worker
		node.DampingFactor = state.DampingFactor
		node.Other = state.Other
		node.UpperLayer = &masterConn
	}

	// Creating gRPC server
	lis, err := net.Listen("tcp", pkg.Url(&conn))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	services.RegisterNodeServer(server, &NodeServerImpl{Node: &node})
	log.Printf("Server listening at %s", pkg.Url(&conn))
	// Running gRPC server in a goroutine
	go func() {
		if err = server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v\n", err)
		}
	}()
	// TODO: connect to queue
	// TODO: Node update loop
}
