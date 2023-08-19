package main

import (
	"context"

	"github.com/lioia/distributed-pagerank/lib"
)

type Layer2NodeServerImpl struct {
	Node *Layer2Node
	lib.UnimplementedLayer2NodeServer
}

func (s *Layer2NodeServerImpl) HealthCheck(context.Context, *lib.Empty) (*lib.Empty, error) {
	return &lib.Empty{}, nil
}

func (s *Layer2NodeServerImpl) ComputeMap(_ context.Context, in *lib.SubGraph) (*lib.MapContributions, error) {
	message := &lib.MapContributions{}
	for _, node := range in.Graph {
		contributions := node.Map()
		for id, v := range contributions {
			message.Contribution[id] += v
		}

	}
	return message, nil
}

func (s *Layer2NodeServerImpl) ComputeReduce(_ context.Context, in *lib.Sum) (*lib.Rank, error) {
	rank := &lib.Rank{
		ID:   in.Node.ID,
		Rank: in.DampingFactor*in.Sum + (1-in.DampingFactor)*in.Node.EValue,
	}
	return rank, nil
}
