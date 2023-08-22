package main

import (
	"context"
	"fmt"

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

func (s *NodeServerImpl) UploadGraph(_ context.Context, in *services.GraphUpload) (*services.MapIntDouble, error) {
	if s.Node.Role != pkg.Master {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.UpperLayer)
	}
	// TODO: if Other is empty, calculate ranks and return to client
	// otherwise, return nil and switch to Map phase
	return nil, nil
}
