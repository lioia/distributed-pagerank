package node

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type NodeServerImpl struct {
	Node *Node
	proto.UnimplementedNodeServer
}

// From worker to master node to check if the master node is still alive
// NOTE: maybe this won't work on new state update method noted in utils/queue.go#L59
func (s *NodeServerImpl) HealthCheck(_ context.Context, in *wrapperspb.StringValue) (*proto.Health, error) {
	var health *proto.Health
	// Check if the contacting node is known to the master
	// A worker node could have been removed from the Others array
	// if it didn't respond to a StateUpdate
	found := false
	for _, v := range s.Node.State.Others {
		if v == in.Value {
			found = true
			break
		}
	}
	if found {
		// Node was found -> last state update went correctly
		health.Value = &proto.Health_Empty{}
	} else {
		// Node was not found -> worker does not have last state update
		s.Node.State.Others = append(s.Node.State.Others, in.Value)
		health.Value = &proto.Health_State{
			State: s.Node.State,
		}
	}
	return health, nil
}

// From master to worker nodes to keep the master node shared on specific events
func (s *NodeServerImpl) StateUpdate(_ context.Context, in *proto.State) (*emptypb.Empty, error) {
	s.Node.State = in
	return &emptypb.Empty{}, nil
}

// From new node to master node
func (s *NodeServerImpl) NodeJoin(_ context.Context, in *wrapperspb.StringValue) (*proto.Join, error) {
	// Worker cannot do this operation
	if s.Node.Role == Worker {
		return &proto.Join{}, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.Master)
	}
	s.Node.State.Others = append(s.Node.State.Others, in.Value)
	constants := proto.Join{
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		State:       s.Node.State,
	}
	return &constants, nil
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
