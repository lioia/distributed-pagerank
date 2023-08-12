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
