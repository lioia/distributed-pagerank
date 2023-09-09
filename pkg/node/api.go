package node

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/pkg/graph"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ApiServerImpl struct {
	Node   *Node     // Node state
	Status chan bool // Client state
	proto.UnimplementedAPIServer
}

func (s *ApiServerImpl) GraphUpload(_ context.Context, in *proto.Configuration) (*emptypb.Empty, error) {
	graph, err := graph.LoadGraphResource(in.Graph)
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("Failed to load graph: %v", err)
	}
	s.Node.State.Client = in.Connection
	s.Node.State.C = in.C
	s.Node.State.Threshold = in.Threshold
	s.Node.State.Graph = graph
	return &emptypb.Empty{}, nil
}

func (s *ApiServerImpl) Results(_ context.Context, in *proto.Result) (*emptypb.Empty, error) {
	fmt.Println("Received results:")
	for id, v := range in.Values {
		fmt.Printf("Node %d with rank %f\n", id, v)
	}
	// Close client
	s.Status <- true
	return &emptypb.Empty{}, nil
}
