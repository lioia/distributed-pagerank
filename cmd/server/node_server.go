package main

import (
	"context"

	"github.com/lioia/distributed-pagerank/pkg/services"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NodeServerImpl struct {
	Node *Node
	services.UnimplementedNodeServer
}

func (s *NodeServerImpl) HealthCheck(context.Context, *emptypb.Empty) (*services.State, error) {
	state := services.State{
		Phase:         int32(s.Node.Phase),
		Role:          int32(s.Node.Role),
		DampingFactor: s.Node.DampingFactor,
		Other:         s.Node.Other,
	}
	return &state, nil
}
