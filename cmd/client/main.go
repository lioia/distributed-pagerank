package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
)

var api string        // Connection string of the server
var file string       // Graph file
var c float64         // C-parameter
var threshold float64 // Convergence Threshold

func init() {
	flag.StringVar(&api, "api", "", "API Server Connection")
	flag.StringVar(&file, "file", "", "Graph file")
	flag.Float64Var(&c, "c", 0, "C Parameter")
	flag.Float64Var(&threshold, "threshold", 0, "Convergence Threshold")
}

func main() {
	flag.Parse()
	if api == "" {
		api = pkg.ReadFromStdinAndFail("Enter API Server Connection: ")
	}
	if file == "" {
		file = pkg.ReadFromStdinAndFail("Enter path to the file describing the graph: ")
	}
	if c == 0 {
		cString := pkg.ReadFromStdinAndFail("Enter c-value: ")
		cValue, err := strconv.ParseFloat(cString, 64)
		pkg.FailOnError("Provided c-value is not a valid number", err)
		c = cValue
	}
	if threshold == 0 {
		thresholdString := pkg.ReadFromStdinAndFail("Enter threshold value: ")
		thresholdValue, err := strconv.ParseFloat(thresholdString, 64)
		pkg.FailOnError("Provided threshold value is not a valid number", err)
		threshold = thresholdValue
	}

	bytes, err := os.ReadFile(file)
	pkg.FailOnError("Failed to read file", err)

	lis, err := net.Listen("tcp", ":0")
	pkg.FailOnError("Failed to listen", err)

	server, err := pkg.ApiCall(api)
	pkg.FailOnError("Failed to call server", err)

	graph := proto.GraphUpload{
		From:      lis.Addr().String(),
		C:         c,
		Threshold: threshold,
		Contents:  bytes,
	}
	results, err := server.Client.SendGraph(server.Ctx, &graph)
	pkg.FailOnError("API Server error", err)
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
