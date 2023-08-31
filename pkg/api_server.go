package pkg

import (
	"context"
	"fmt"
	"log"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ApiServerImpl struct {
	Node *Node
	proto.UnimplementedApiServer
}

// From client to master
func (s *ApiServerImpl) SendGraph(_ context.Context, in *proto.GraphUpload) (*emptypb.Empty, error) {
	empty := &emptypb.Empty{}
	log.Printf("Received from: %s\n", in.From)
	if s.Node.Role != Master {
		return empty, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.Master)
	}
	graph, err := LoadGraphFromBytes(in.Contents)
	if err != nil {
		return empty, fmt.Errorf("Could not load graph: %v", err)
	}
	// No other node in the network -> calculating PageRank on this node
	if len(s.Node.State.Others) == 0 {
		SingleNodePageRank(graph, in.C, in.Threshold)
		// Send results to client
		client, err := ApiCall(in.From)
		if err != nil {
			return empty, err
		}
		_, err = client.Client.ReceiveResults(client.Ctx, graph)
		if err != nil {
			return empty, err
		}
		return empty, nil
	}
	// Saving computation information
	s.Node.State.Client = in.From
	s.Node.State.Graph = graph
	s.Node.C = in.C
	s.Node.Threshold = in.Threshold
	// Divide the graph in message and publish to queue
	err = s.Node.WriteGraphToQueue()
	// Send state update to worker nodes
	s.Node.masterSendUpdateToWorkers()
	// Graph was successfully uploaded and computation has started
	return empty, err
}

// From master to client
func (s *ApiServerImpl) ReceiveResults(_ context.Context, in *proto.Graph) (*emptypb.Empty, error) {
	fmt.Println("Received results:")
	for id, v := range in.Graph {
		fmt.Printf("%d -> %f\n", id, v.Rank)
	}
	return &emptypb.Empty{}, nil
}
