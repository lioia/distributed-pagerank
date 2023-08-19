package main

import (
	"context"
	"fmt"

	"github.com/lioia/distributed-pagerank/lib"
)

type MasterNodeServerImpl struct {
	Node *MasterNode
	lib.UnimplementedMasterNodeServer
}

func (s *MasterNodeServerImpl) HealthCheck(ctx context.Context, in *lib.Empty) (*lib.Empty, error) {
	return &lib.Empty{}, nil
}

func (s *MasterNodeServerImpl) ProcessNewNode(ctx context.Context, in *lib.ConnectionInfo) (*lib.Info, error) {
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

// Return ranks, nil if there is only the master node,
// otherwise returns nil, nil to indicate that the request has been _accepted_
func (s *MasterNodeServerImpl) ProcessGraph(ctx context.Context, in *lib.GraphUpload) (*lib.Ranks, error) {
	var ranks *lib.Ranks
	graph := make(lib.Graph)
	if err := graph.LoadFromBytes(in.Contents); err != nil {
		return ranks, err
	}
	// No other nodes in the network, compute everything by itself
	// and returns the ranks to the client
	if len(s.Node.Layer1s) == 0 {
		// TODO: 0.85 and 0.0001 should be configurable variables
		graph.SingleNodePageRank(0.85, 0.0001)
		i := 0
		for id, node := range graph {
			ranks.Ranks[i] = &lib.Rank{ID: id, Rank: node.Rank}
			i += 1
		}
		return ranks, nil
	}
	// Save computation information
	s.Node.Client = in.From // used to call client to send ranks
	s.Node.Graph = graph
	s.Node.Phase = Map // Map Phase
	// # nodes to send to worker
	graphNodesPerNetworkNodes := len(graph) / len(s.Node.Layer1s)
	// Divide graph into multiple subgraphs
	subGraphs := make([]lib.Graph, len(s.Node.Layer1s))
	index := 0
	for id, node := range graph {
		subGraphs[index/graphNodesPerNetworkNodes][id] = node
		index += 1
	}
	// For each Layer 1 node, send subgraph
	for i := 0; i < len(s.Node.Layer1s); i++ {
		layer1 := s.Node.Layer1s[i]
		clientUrl := fmt.Sprintf("%s:%d", layer1.Address, layer1.Port)
		clientInfo, err := lib.Layer1ClientCall(clientUrl)
		// FIXME: error handling
		if err != nil {
			return ranks, err
		}
		subGraph := lib.SubGraph{Graph: subGraphs[i]}
		_, err = clientInfo.Client.ReceiveGraph(clientInfo.Ctx, &subGraph)
		// FIXME: error handling
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
