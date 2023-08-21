package main

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lioia/distributed-pagerank/lib"
)

func (_ *MasterNode) Init(*lib.Info) error {
	return nil
}

func (n *MasterNode) Update() error {
	var err error
	switch n.Phase {
	case Map:
		err = n.Map()
	case Convergence:
		err = n.Convergence()
	}
	return err
}

// Check if layer 1 nodes are still present
// Layer 1 nodes repond with the # layer 2 nodes and the current phase
// If a layer 1 node doesn't respond, it is assumed that it crashed ->
// depending on Phase action are taken
func (n *MasterNode) StateCheck() error {
	for i, layer1 := range n.Layer1s {
		clientUrl := fmt.Sprintf("%s:%d", layer1.Address, layer1.Port)
		clientInfo, err := lib.Layer1ClientCall(clientUrl)
		// FIXME: error handling
		if err != nil {
			return err
		}
		state, err := clientInfo.Client.HealthCheck(clientInfo.Ctx, nil)
		if err != nil {
			// TODO: assuming crash, do something:
			// - remove from Layer 1 Nodes
			// - contact other layer 1 nodes to inform of changes
			// - do something based on phase
			return err
		}
		n.NumberOfNodes[i] = state.NumberOfLayer2
		// TODO: probably some checks are necessary
		n.Phase = Phase(state.Phase)
	}
	return nil
}

func (n *MasterNode) Map() error {
	// No other nodes in the network, compute everything by itself
	// and returns the ranks to the client
	if len(n.Layer1s) == 0 {
		n.Graph.SingleNodePageRank(n.DampingFactor, n.Threshold)
		return n.SendDataToClient()
	}
	// # nodes to send to worker
	graphNodesPerNetworkNodes := len(n.Graph) / len(n.Layer1s)
	// Divide graph into multiple subgraphs
	subGraphs := make([]lib.Graph, len(n.Layer1s))
	index := 0
	for id, node := range n.Graph {
		subGraphs[index/graphNodesPerNetworkNodes][id] = node
		index += 1
	}
	// For each Layer 1 node, send subgraph
	for i := 0; i < len(n.Layer1s); i++ {
		layer1 := n.Layer1s[i]
		clientUrl := fmt.Sprintf("%s:%d", layer1.Address, layer1.Port)
		clientInfo, err := lib.Layer1ClientCall(clientUrl)
		// FIXME: error handling
		if err != nil {
			return err
		}
		subGraph := lib.SubGraph{Graph: subGraphs[i]}
		_, err = clientInfo.Client.ReceiveGraph(clientInfo.Ctx, &subGraph)
		// FIXME: error handling
		if err != nil {
			return err
		}
	}
	n.Phase = Convergence
	return nil
}

func (n *MasterNode) Convergence() error {
	if int(n.Counter) == len(n.Layer1s) {
		convergence := 0.0
		for id, newRank := range n.NewRanks {
			oldRank := n.Graph[id].Rank
			convergence += math.Abs(newRank - oldRank)
			// After calculating the convergence value, it can be safely updated
			n.Graph[id].Rank = newRank
		}
		// Does not converge -> iterate
		if convergence > n.Threshold {
			// Start new computation with updated pagerank values
			n.Phase = Map
			return nil
		} else {
			n.SendDataToClient()
			// TODO: converged, send data to client
			n.Phase = Wait
		}
		n.Counter = 0
	}
	return nil
}

func (n *MasterNode) SendDataToClient() error {
	// clientUrl := fmt.Sprintf("%s:%d", n.Client.Address, n.Client.Port)
	// clientInfo, err := lib.Layer1ClientCall(clientUrl)
	// // FIXME: error handling
	// if err != nil {
	// 	return err
	// }
	// contrib := &lib.MapIntDouble{Map: n.MapData}
	// _, err = clientInfo.Client.Respond(clientInfo.Ctx, contrib)
	// // FIXME: error handling
	// if err != nil {
	// 	return err
	// }
	return errors.New("unimplemented")
}

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
func (s *MasterNodeServerImpl) ProcessGraph(ctx context.Context, in *lib.GraphUpload) (*lib.Empty, error) {
	var empty *lib.Empty
	graph := make(lib.Graph)
	if err := graph.LoadFromBytes(in.Contents); err != nil {
		return nil, err
	}
	s.Node.Graph = graph
	s.Node.Client = in.From
	s.Node.Phase = Map

	return empty, nil
}

func (s *MasterNodeServerImpl) SyncRanks(_ context.Context, in *lib.MapIntDouble) (*lib.Empty, error) {
	empty := &lib.Empty{}
	s.Node.Phase = Convergence
	s.Node.NewRanks = in.Map
	return empty, nil
}
