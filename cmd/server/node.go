package main

import (
	"fmt"

	"github.com/lioia/distributed-pagerank/lib"
)

type Node interface {
	Init(info *lib.Info) error
}

// Phase can be treated as an enum
// (iota: the contants in this group, of type Phase, are auto-increment)
type Phase int32

const (
	Wait            Phase = iota // No graph uploaded, node wait for instruction
	Map                          // Map Computation
	Collect                      // Layer 1 nodes groups and sync map phase results
	Reduce                       // Reduce Computation
	Convergence                  // Layer 1 nodes decices what to do next
	Synchronization              // Convergence failed, updating ranks from other nodes
)

type BaseNode struct {
	Layer   int32
	Address string
	Port    int32
	Graph   lib.Graph
	Phase   Phase
}

type MasterNode struct {
	BaseNode
	Layer1s       []*lib.ConnectionInfo // Connection info for layer 1nodes
	NumberOfNodes []int32               // # nodes for every layer 1
	Client        *lib.ConnectionInfo   // Client connection info
}

type Layer1Node struct {
	BaseNode                        // Base node information
	FirstNode *lib.ConnectionInfo   // Connection info of first node
	Layer1s   []*lib.ConnectionInfo // Connection info for other first layer nodes
	Layer2s   []*lib.ConnectionInfo // Connection info for second layer nodes
	SubGraphs []lib.Graph           // Graph associated with layer 2 node
	MapData   map[int32]float64     // Partial sums derived from map phase
	Counter   int32                 // Number of responses received
}

type Layer2Node struct {
	BaseNode                     // Base node information
	Layer1   *lib.ConnectionInfo // Assigned first layer connection info
}

func (_ *MasterNode) Init(*lib.Info) error {
	return nil
}

func (n *Layer1Node) Init(info *lib.Info) error {
	for _, v := range info.GetLayer1S() {
		// Save information on the other layer 1 nodes
		n.Layer1s = append(n.Layer1s, v)
		// Contact other layer 1 nodes
		layer1Url := fmt.Sprintf("%s:%d", v.Address, v.Port)
		clientInfo, err := lib.Layer1ClientCall(layer1Url)
		// FIXME: error handling
		if err != nil {
			return err
		}
		announceMsg := lib.AnnounceMessage{
			LayerNumber: 1,
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
	}
	return nil
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
