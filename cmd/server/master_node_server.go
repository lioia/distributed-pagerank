package main

import (
	"context"

	"github.com/lioia/distributed-pagerank/lib"
)

type MasterNodeServerImpl struct {
	Node *MasterNode
	lib.UnimplementedMasterNodeServer
}

func (s *MasterNodeServerImpl) HealthCheck(ctx context.Context, in *lib.Empty) (*lib.Empty, error) {
	return &lib.Empty{}, nil
}

func (s *MasterNodeServerImpl) GetInfo(ctx context.Context, in *lib.ConnectionInfo) (*lib.Info, error) {
	var info lib.Info
	// Dynamic Node Placement
	info.LayerNumber = 1
	foundNoSubNodes := false
	// New node will be placed as a sub-node to a layer 1 node with 0 sub-nodes
	for _, v := range s.Node.NumberOfNodes {
		if v == 0 {
			foundNoSubNodes = true
			break
		}
	}
	// TODO: 3 should be a configurable variable
	if foundNoSubNodes || len(s.Node.NumberOfNodes) > 3 {
		info.LayerNumber = 2
	}
	if info.LayerNumber == 1 { // Layer 1 Node
		// Sending already present layer 1 nodes in the network
		info.Layer1S = s.Node.Layer1s
		// Add node requesting info to list of layer1s
		s.Node.Layer1s = append(s.Node.Layer1s, in)
	} else { // Layer 2 Node
		// Find the layer 1 node with the least number of nodes
		// NOTE: this search could be moved to the previous for loop (L-17)
		var assigned int
		var minNumOfNodes int32 = 1<<31 - 1 // max int32 value
		for i, v := range s.Node.NumberOfNodes {
			if minNumOfNodes > v {
				minNumOfNodes = v
				assigned = i
			}
		}
		info.Assigned = s.Node.Layer1s[assigned]
		s.Node.NumberOfNodes[assigned] += 1
	}
	return &info, nil

}

func (s *MasterNodeServerImpl) UploadGraph(ctx context.Context, in *lib.GraphFile) (*lib.Ranks, error) {
	var ranks *lib.Ranks
	graph := make(lib.Graph)
	if err := graph.LoadFromBytes(in.Contents); err != nil {
		return ranks, err
	}
	// No other nodes in the network, compute everything by itself
	if len(s.Node.Layer1s) == 0 {
		// TODO: 0.85 and 0.0001 should be configurable variables
		graph.SingleNodePageRank(0.85, 0.0001)
		i := 0
		for id, node := range graph {
			ranks.ID[i] = id
			ranks.Ranks[i] = node.Rank
			i += 1
		}
		return ranks, nil
	}
	// s.Node.Graph = graph
	// graphNodesPerNetworkNodes := len(graph) / len(s.Node.Layer1s)
	// subGraphs := make([]lib.Graph, len(s.Node.Layer1s))
	// index := 0
	// for id, node := range graph {
	// 	subGraphs[index/graphNodesPerNetworkNodes][id] = node
	// 	index += 1
	// }
	// for i := 0; i < len(s.Node.Layer1s); i++ {
	// 	layer1 := s.Node.Layer1s[i]
	// 	subGraph := subGraphs[i]
	// 	clientUrl := fmt.Sprintf("%s:%d", layer1.Address, layer1.Port)
	// 	clientInfo, err := lib.ClientCall(clientUrl)
	// 	// FIXME: error handling
	// 	if err != nil {
	// 		return ranks, err
	// 	}
	// 	// TODO: send
	// }

	return ranks, nil
}
