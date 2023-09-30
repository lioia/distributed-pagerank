package node

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/pkg/graph"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ApiServerImpl struct {
	Node  *Node             // Node state
	Ranks chan *proto.Ranks // Client state
	proto.UnimplementedAPIServer
}

func (s *ApiServerImpl) GraphUpload(_ context.Context, in *proto.Configuration) (*emptypb.Empty, error) {
	var err error
	g := make(map[int32]*proto.GraphNode)
	if state := in.GetGraph(); state != "" {
		// Graph URL was provided, downloading and parsing the graph
		g, err = graph.LoadGraphResource(state)
		if err != nil {
			return &emptypb.Empty{}, fmt.Errorf("Failed to load graph: %v", err)
		}

	} else if state := in.GetRandomGraph(); state != nil {
		// Random graph config was provided, generaring the graph
		g = graph.Generate(state.NumberOfNodes, state.MaxNumberOfEdges)
	}
	s.Node.State.Client = in.Connection
	s.Node.State.C = in.C
	s.Node.State.Threshold = in.Threshold
	s.Node.State.Graph = g
	return &emptypb.Empty{}, nil
}

func (s *ApiServerImpl) Results(_ context.Context, in *proto.Ranks) (*emptypb.Empty, error) {
	s.Ranks <- in
	return &emptypb.Empty{}, nil
}
