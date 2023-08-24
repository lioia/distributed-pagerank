package pkg

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NodeServerImpl struct {
	Node *Node
	proto.UnimplementedNodeServer
}

func (s *NodeServerImpl) StateUpdate(context.Context, *emptypb.Empty) (*proto.State, error) {
	state := proto.State{
		Phase:  int32(s.Node.Phase),
		Role:   int32(s.Node.Role),
		Graph:  s.Node.Graph,
		Others: s.Node.Others,
	}
	return &state, nil
}

func (s *NodeServerImpl) NodeJoin(context.Context, *emptypb.Empty) (*proto.Constants, error) {
	if s.Node.Role == Worker {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.UpperLayer)
	}
	constants := proto.Constants{
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		C:           s.Node.C,
		Threshold:   s.Node.Threshold,
	}
	return &constants, nil
}

func (s *NodeServerImpl) UploadGraph(_ context.Context, in *proto.GraphUpload) (*proto.Graph, error) {
	if s.Node.Role != Master {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.UpperLayer)
	}
	graph, err := LoadGraphFromBytes(in.Contents)
	if err != nil {
		return nil, fmt.Errorf("Could not load graph: %v", err)
	}
	// No other node in the network -> calculating PageRank on this node
	if len(s.Node.Others) == 0 {
		SingleNodePageRank(graph, s.Node.C, s.Node.Threshold)
		return graph, nil
	}
	err = s.Node.WriteGraphToQueue()
	// Graph was successfully uploaded and computation has started
	return nil, err
}
