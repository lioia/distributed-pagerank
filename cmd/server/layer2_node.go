package main

import (
	"context"
	"fmt"

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

func (n *Layer2Node) Init(info *lib.Info) error {
	n.Layer1 = info.GetAssigned()
	layer1Url := fmt.Sprintf("%s:%d", n.Layer1.Address, n.Layer1.Port)
	clientInfo, err := lib.Layer1ClientCall(layer1Url)
	// FIXME: error handling
	if err != nil {
		return err
	}
	announceMsg := lib.AnnounceMessage{
		LayerNumber: 2,
		Connection: &lib.ConnectionInfo{
			Address: n.Address,
			Port:    n.Port,
		},
	}
	_, err = clientInfo.Client.Announce(clientInfo.Ctx, &announceMsg)
	// FIXME: error handling
	if err != nil {
		return err
	}
	return nil
}

func (n *Layer2Node) Update() error {
	// TODO: implement what the layer2 node has to do
	return nil
}
