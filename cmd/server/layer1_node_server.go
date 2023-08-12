package main

import (
	"context"
	"errors"

	"github.com/lioia/distributed-pagerank/lib"
)

type Layer1NodeServerImpl struct {
	Node *Layer1Node
	lib.UnimplementedLayer1NodeServer
}

func (s *Layer1NodeServerImpl) HealthCheck(context.Context, *lib.Empty) (*lib.Empty, error) {
	return &lib.Empty{}, nil
}

func (s *Layer1NodeServerImpl) Announce(_ context.Context, in *lib.AnnounceMessage) (*lib.Empty, error) {
	if in.LayerNumber == 1 {
		s.Node.Layer1s = append(s.Node.Layer1s, in.Connection)
	} else if in.LayerNumber == 2 {
		s.Node.Layer2s = append(s.Node.Layer2s, in.Connection)
	} else {
		return &lib.Empty{}, errors.New("invalid layer number")
	}
	return &lib.Empty{}, nil
}
