package main

import (
	"fmt"
	"net"

	"github.com/lioia/distributed-pagerank/pkg/node"
	"github.com/lioia/distributed-pagerank/pkg/utils"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
)

func main() {
	host := utils.ReadStringEnvVarOr("HOST", "")
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:0", host))
	utils.FailOnError("Failed to listen for node server", err)
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port
	if host == "" {
		host = lis.Addr().(*net.TCPAddr).IP.String()
	}

	api := utils.ReadStringFromStdin("Enter the API url: ")
	c := utils.ReadFloat64FromStdin("Enter c-value [in range (0.0..1.0)]: ")
	threshold := utils.ReadFloat64FromStdin("Enter threshold [in range (0.0..1.0)]: ")
	graph := utils.ReadStringFromStdin("Enter graph file [network resource]: ")

	configuration := &proto.Configuration{
		Connection: fmt.Sprintf("%s:%d", host, port),
		C:          c,
		Threshold:  threshold,
		Graph:      graph,
	}
	client, err := utils.ApiCall(api)
	utils.FailOnError("Failed to connect to client", err)
	defer client.Close()
	_, err = client.Client.GraphUpload(client.Ctx, configuration)
	utils.FailOnError("Failed to send configuration to server", err)

	fmt.Println("Waiting for computation to finish")
	server := grpc.NewServer()
	proto.RegisterAPIServer(server, &node.ApiServerImpl{})
	err = server.Serve(lis)
	utils.FailOnError("Failed to serve", err)
}
