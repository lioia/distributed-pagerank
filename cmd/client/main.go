package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
)

var api string  // Connection string of the server
var file string // Graph file

func init() {
	flag.StringVar(&api, "api", "127.0.0.1:1234", "API Server Connection")
	flag.StringVar(&file, "file", "graph.txt", "Graph file")
}

func main() {
	flag.Parse()

	bytes, err := os.ReadFile(file)
	pkg.FailOnError("Failed to read file", err)

	lis, err := net.Listen("tcp", "0")
	pkg.FailOnError("Failed to listen", err)

	server, err := pkg.ApiCall(api)
	pkg.FailOnError("Failed to call server", err)

	graph := proto.GraphUpload{
		From:     lis.Addr().String(),
		Contents: bytes,
	}
	results, err := server.Client.SendGraph(server.Ctx, &graph)
	pkg.FailOnError("Server error", err)
	if results != nil {
		fmt.Println("Received results:")
		for id, v := range results.Graph {
			fmt.Printf("%d -> %f\n", id, v.Rank)
		}
		os.Exit(0)
	}

	// Starting gRPC server to receive results
	s := grpc.NewServer()
	proto.RegisterApiServer(s, &pkg.ApiServerImpl{})
	log.Printf("Starting api server at %s\n", lis.Addr().String())
	err = s.Serve(lis)
	pkg.FailOnError("Failed to serve", err)
}
