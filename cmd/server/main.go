package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/lioia/distributed-pagerank/lib"
	"google.golang.org/grpc"
)

var port int              // Port where a new node will be created
var firstAddress string   // Expected address of the first node of the network
var firstPort int         // Expected port of the first node
var dampingFactor float64 // PageRank `c` parameter
var threshold float64     // PageRank threshold

func init() {
	flag.IntVar(&port, "port", 1234, "Node Port number")
	flag.StringVar(&firstAddress, "firstAddress", "127.0.0.1", "First Node Address")
	flag.IntVar(&firstPort, "firstPort", 1234, "First Node Port number")
	flag.Float64Var(&dampingFactor, "dampingFactor", 0.85, "Damping Factor (c)")
	flag.Float64Var(&threshold, "threshold", 0.0001, "Threshold")
}

func main() {
	flag.Parse()

	// TODO: read configuration file

	// Get local address from network interfaces
	address, err := lib.GetAddressFromInterfaces()
	if err != nil {
		log.Fatalf("unable to determine local address: %v\n", err)
	}
	var node Node
	baseNode := BaseNode{
		Layer:   0,
		Address: address,
		Port:    int32(port),
	}
	s := grpc.NewServer()

	// First Node Url
	firstUrl := fmt.Sprintf("%s:%d", firstAddress, firstPort)

	// Client towards first node
	clientInfo, err := lib.MasterClientCall(firstUrl)
	if err != nil {
		panic(err)
	}

	// Check whether the first node is present
	_, err = clientInfo.Client.HealthCheck(clientInfo.Ctx, nil)
	if err != nil {
		// There is no node at the address
		// This node is the first node of the network
		fmt.Printf("No node found at %s. Running at %s:%d\n", firstUrl, address, firstPort)
		// TODO: do something
		baseNode.Port = int32(firstPort)
		node = &MasterNode{
			BaseNode: baseNode,
		}
		lib.RegisterMasterNodeServer(s, &MasterNodeServerImpl{
			Node: (node).(*MasterNode),
		})
	} else {
		// There is a node in the network
		// Contacting the first node to get the role
		selfConnInfo := lib.ConnectionInfo{
			Address: address,
			Port:    int32(port),
		}
		info, err := clientInfo.Client.ProcessNewNode(clientInfo.Ctx, &selfConnInfo)
		if err != nil {
			panic(err)
		}
		layer := info.GetLayerNumber()
		baseNode.Layer = layer
		if layer == 1 {
			node = &Layer1Node{
				BaseNode: baseNode,
				FirstNode: &lib.ConnectionInfo{
					Address: firstAddress,
					Port:    int32(firstPort),
				},
			}
			lib.RegisterLayer1NodeServer(s, &Layer1NodeServerImpl{
				Node: (node).(*Layer1Node),
			})
		} else if layer == 2 {
			node = &Layer2Node{BaseNode: baseNode}
			lib.RegisterLayer2NodeServer(s, &Layer2NodeServerImpl{
				Node: (node).(*Layer2Node),
			})
		}
		err = node.Init(info)
		// FIXME: error handling
		if err != nil {
			panic(err)
		}
	}
	clientInfo.CancelFunc()
	clientInfo.Conn.Close()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
