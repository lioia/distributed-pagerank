package main

import (
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
	Wait        Phase = iota // No graph uploaded, node wait for instruction
	Map                      // Map Computation
	Collect                  // Layer 1 nodes groups and sync map phase results
	Reduce                   // Reduce Computation
	Convergence              // Layer 1 nodes decices what to do next
	Sync                     // Convergence failed, updating ranks from other nodes
)

type BaseNode struct {
	Layer         int32
	Address       string
	Port          int32
	Graph         lib.Graph
	Phase         Phase
	DampingFactor float64
}

// Implemented in master_node.go
type MasterNode struct {
	BaseNode
	Layer1s       []*lib.ConnectionInfo // Connection info for layer 1nodes
	NumberOfNodes []int32               // # nodes for every layer 1
	Client        *lib.ConnectionInfo   // Client connection info
}

// Implemented in layer1_node.go
type Layer1Node struct {
	BaseNode                         // Base node information
	FirstNode  *lib.ConnectionInfo   // Connection info of first node
	Layer1s    []*lib.ConnectionInfo // Connection info for other first layer nodes
	Layer2s    []*lib.ConnectionInfo // Connection info for second layer nodes
	SubGraphs  []lib.Graph           // Graph associated with layer 2 node
	Counter    int32                 // Number of responses received
	MapData    map[int32]float64     // Partial sums derived from map phase
	ReduceData map[int32]float64     // Updated PageRank
}

// Implemented in layer2_node.go
type Layer2Node struct {
	BaseNode                     // Base node information
	Layer1   *lib.ConnectionInfo // Assigned first layer connection info
}
