package main

import (
	"fmt"
	"sync"

	"github.com/lioia/distributed-pagerank/lib"
)

type Node interface {
	Init(info *lib.Info) error
	Update() error
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

func (n *MasterNode) Update() error {
	// TODO: implement what the master node has to do
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

func (n *Layer1Node) Update() error {
	// TODO: implement what the layer 1 node has to do
	switch n.Phase {
	// Send data to layer 2 nodes and wait for results (in goroutines)
	case Map:
		n.Map()
		// TODO: case Collect: Send data to layer 1 nodes and wait for their data
	}
	return nil
}

func (n *Layer1Node) Map() {
	var wg sync.WaitGroup
	errored := make(chan int) // -1: no errors; >= 0 i-th layer 2 error
	// For each layer 2 node
	for i, layer2 := range n.Layer2s {
		wg.Add(1)
		// Create goroutine, send subgraph and wait for results
		go func(i int, layer2 *lib.ConnectionInfo) {
			defer wg.Done()
			subGraph := n.SubGraphs[i]
			clientUrl := fmt.Sprintf("%s:%d", layer2.Address, layer2.Port)
			clientInfo, err := lib.Layer2ClientCall(clientUrl)
			// FIXME: error handling
			if err != nil {
				errored <- i
				return
			}
			message := lib.SubGraph{Graph: subGraph}
			maps, err := clientInfo.Client.ComputeMap(clientInfo.Ctx, &message)
			// FIXME: error handling
			if err != nil {
				errored <- i
				return
			}
			for id, v := range maps.GetContribution() {
				n.MapData[id] += v
			}
			n.Counter += 1
			errored <- -1
		}(i, layer2)
	}
	for i := range errored {
		// i-th layer 2 node errored
		if i != -1 {
			// Remove from network (assuming crash)
			n.Layer2s = append(n.Layer2s[:i], n.Layer2s[i+1:]...)
			// Calculating Map in this node
			for _, node := range n.SubGraphs[i] {
				contributions := node.Map()
				for id, v := range contributions {
					n.MapData[id] += v
				}
			}
		}
	}
	wg.Wait()
	// Map phase completed, go to Collect phase
	n.Counter = 0
	n.Phase = Collect
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
