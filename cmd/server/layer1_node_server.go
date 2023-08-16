package main

import (
	"context"
	"errors"
	"fmt"

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

func (s *Layer1NodeServerImpl) ReceiveGraph(_ context.Context, in *lib.SubGraph) (*lib.Empty, error) {
	empty := &lib.Empty{}
	s.Node.MapData = make(map[int32]float64)
	// No layer 2 nodes, computing map by itself and switch to Collect phase
	if len(s.Node.Layer2s) == 0 {
		for _, node := range in.Graph {
			contributions := node.Map()
			for id, v := range contributions {
				s.Node.MapData[id] += v
			}
		}
		s.Node.Phase = Collect
		return empty, nil
	}
	// Save information and set to Map phase
	s.Node.Graph = in.Graph
	s.Node.Phase = Map
	s.Node.SubGraphs = make([]lib.Graph, len(s.Node.Layer2s))
	// # nodes to send to layer 2 network node
	graphNodesPerNetworkNodes := len(in.Graph) / len(s.Node.Layer2s)
	// Divide graph into multiple subgraphs
	index := 0
	for id, node := range in.Graph {
		s.Node.SubGraphs[index/graphNodesPerNetworkNodes][id] = node
		index += 1
	}
	// For each layer 2 node, send subgraph
	// TODO: switch to main function and goroutines
	for i := 0; i < len(s.Node.Layer2s); i++ {
		layer2 := s.Node.Layer2s[i]
		subGraph := s.Node.SubGraphs[i]
		clientUrl := fmt.Sprintf("%s:%d", layer2.Address, layer2.Port)
		clientInfo, err := lib.Layer2ClientCall(clientUrl)
		// FIXME: error handling
		if err != nil {
			return nil, err
		}
		message := lib.SubGraph{Graph: subGraph}
		contributions, err := clientInfo.Client.ComputeMap(clientInfo.Ctx, &message)
		// FIXME: error handling
		if err != nil {
			return nil, err
		}
		for id, v := range contributions.GetContribution() {
			s.Node.MapData[id] += v
		}
	}

	return empty, nil
}
