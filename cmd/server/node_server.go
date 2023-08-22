package main

import (
	"context"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/pkg/services"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NodeServerImpl struct {
	Node *pkg.Node
	services.UnimplementedNodeServer
}

func (s *NodeServerImpl) HealthCheck(context.Context, *emptypb.Empty) (*services.State, error) {
	state := services.State{
		Phase:       int32(s.Node.Phase),
		Role:        int32(s.Node.Role),
		C:           s.Node.C,
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		Other:       s.Node.Other,
	}
	return &state, nil
}
