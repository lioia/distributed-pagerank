package node

import (
	"context"

	"github.com/lioia/distributed-pagerank/pkg/utils"
	"github.com/lioia/distributed-pagerank/proto"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type NodeServerImpl struct {
	Node *Node
	proto.UnimplementedNodeServer
}

// From worker to master node to check if the master node is still alive
func (s *NodeServerImpl) HealthCheck(_ context.Context, in *wrapperspb.StringValue) (*proto.Health, error) {
	// utils.ServerLog("HealthCheck")
	s.Node.mu.Lock()
	health := &proto.Health{}
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
		id, _ := gonanoid.New()
		_, ok := s.Node.State.Others[id]
		for ok || id == s.Node.Id {
			// ID already assigned to node; generating new one
			id, _ = gonanoid.New()
			_, ok = s.Node.State.Others[id]
		}
		if s.Node.State.Others == nil {
			s.Node.State.Others = make(map[string]string)
		}
		s.Node.State.Others[id] = in.Value
		// s.Node.State.Others = append(s.Node.State.Others, in.Value)VgVg
		health.Value = &proto.Health_State{
			State: s.Node.State,
		}
	}
	s.Node.mu.Unlock()
	return health, nil
}

// From master to worker nodes to keep the master node shared on specific events
func (s *NodeServerImpl) StateUpdate(_ context.Context, in *proto.State) (*emptypb.Empty, error) {
	utils.ServerLog("StateUpdate")
	s.Node.State = in
	return &emptypb.Empty{}, nil
}

// From master to worker nodes to keep the master node shared on specific events
func (s *NodeServerImpl) OtherStateUpdate(_ context.Context, in *proto.OtherState) (*emptypb.Empty, error) {
	utils.ServerLog("OtherStateUpdate")
	s.Node.State.Others = in.Connections
	return &emptypb.Empty{}, nil
}

// From new node to master node
func (s *NodeServerImpl) NodeJoin(_ context.Context, in *wrapperspb.StringValue) (*proto.Join, error) {
	utils.ServerLog("NodeJoin: %s", in.Value)
	s.Node.mu.Lock()
	id, _ := gonanoid.New()
	_, ok := s.Node.State.Others[id]
	for ok || id == s.Node.Id {
		// ID already assigned to node; generating new one
		id, _ = gonanoid.New()
		_, ok = s.Node.State.Others[id]
	}
	utils.NodeLog("master", "Assigning %s to %s", id, in.Value)
	s.Node.State.Others[id] = in.Value
	constants := proto.Join{
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		State:       s.Node.State,
		Id:          id,
	}
	s.Node.mu.Unlock()
	go masterSendOtherStateUpdate(s.Node)
	return &constants, nil
}

// From worker node to worker nodes to announce a new candidacy
func (s *NodeServerImpl) MasterCandidate(_ context.Context, in *proto.Candidacy) (*wrapperspb.BoolValue, error) {
	utils.ServerLog("MasterCandidate")
	msg := "refused"
	ack := wrapperspb.BoolValue{Value: false}
	if in.Id > s.Node.Candidacy {
		ack.Value = true
		s.Node.Candidacy = in.Id
		s.Node.Master = in.Connection
		msg = "accepted"
	}
	utils.ServerLog("MasterCandidate: %s master candidate from %s", msg, in.Connection)
	return &ack, nil
}
