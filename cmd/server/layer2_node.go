package main

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/lib"
)

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
	// Nothing to do
	return nil
}

func (n *Layer2Node) StateCheck() error {
	// Nothing to do
	return nil
}

type Layer2NodeServerImpl struct {
	Node *Layer2Node
	lib.UnimplementedLayer2NodeServer
}

func (s *Layer2NodeServerImpl) ComputeMap(_ context.Context, in *lib.SubGraph) (*lib.MapIntDouble, error) {
	message := &lib.MapIntDouble{}
	for _, node := range in.Graph {
		contributions := node.Map()
		for id, v := range contributions {
			message.Map[id] += v
		}

	}
	return message, nil
}

func (s *Layer2NodeServerImpl) ComputeReduce(_ context.Context, in *lib.Sums) (*lib.MapIntDouble, error) {
	var ranks lib.MapIntDouble
	for i, v := range in.Nodes {
		ranks.Map[v.ID] = in.DampingFactor*in.Sums[i] + (1-in.DampingFactor)*v.EValue
	}
	return &ranks, nil
}
