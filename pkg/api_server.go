package pkg

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ApiServerImpl struct {
	Node *Node
	proto.UnimplementedApiServer
}

// From client to master
func (s *ApiServerImpl) SendGraph(_ context.Context, in *proto.GraphUpload) (*proto.Graph, error) {
	if s.Node.Role != Master {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.Master)
	}
	graph, err := LoadGraphFromBytes(in.Contents)
	if err != nil {
		return nil, fmt.Errorf("Could not load graph: %v", err)
	}
	// No other node in the network -> calculating PageRank on this node
	if len(s.Node.State.Others) == 0 {
		SingleNodePageRank(graph, s.Node.C, s.Node.Threshold)
		return graph, nil
	}
	s.Node.State.Client = in.From
	err = s.Node.WriteGraphToQueue()
	s.Node.masterSendUpdateToWorkers()
	// Graph was successfully uploaded and computation has started
	return nil, err
}

// From master to client
func (s *ApiServerImpl) ReceiveResults(_ context.Context, in *proto.Graph) (*emptypb.Empty, error) {
	fmt.Println("Received results:")
	for id, v := range in.Graph {
		fmt.Printf("%d -> %f\n", id, v.Rank)
	}
	return &emptypb.Empty{}, nil
}
