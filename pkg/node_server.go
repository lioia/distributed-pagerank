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

// From worker to master node to check if the master node is still alive
func (s *NodeServerImpl) HealthCheck(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

// From master to worker nodes to keep the master node shared on specific events
func (s *NodeServerImpl) StateUpdate(_ context.Context, in *proto.State) (*emptypb.Empty, error) {
	s.Node.State = in
	return &emptypb.Empty{}, nil
}

// From new node to master node
func (s *NodeServerImpl) NodeJoin(context.Context, *emptypb.Empty) (*proto.Join, error) {
	if s.Node.Role == Worker {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.Master)
	}
	constants := proto.Join{
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		C:           s.Node.C,
		Threshold:   s.Node.Threshold,
		State:       s.Node.State,
	}
	return &constants, nil
}

// From client to master Node
// TODO: maybe this can be moved to another service that shares *Node
func (s *NodeServerImpl) UploadGraph(_ context.Context, in *proto.GraphUpload) (*proto.Graph, error) {
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
	// Graph was successfully uploaded and computation has started
	return nil, err
}

// From worker node to worker nodes to announce a new candidacy
func (s *NodeServerImpl) MasterCandidate(_ context.Context, in *proto.Candidacy) (*proto.Ack, error) {
	var ack proto.Ack
	if s.Node.Candidacy > 0 && s.Node.Candidacy < in.Timestamp {
		// There is already a candidate
		ack.Ack = false
		ack.Candidate = s.Node.Master
	} else {
		// Valid candidacy
		ack.Ack = true
		s.Node.Master = in.Connection
	}
	return &ack, nil
}
