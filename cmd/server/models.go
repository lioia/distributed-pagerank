package main

import "github.com/lioia/distributed-pagerank/pkg/services"

// Phase can be treated as an enum
// (iota: the contants in this group, of type Phase, are auto-increment)
type Phase int32

const (
	Wait        Phase = iota // No graph uploaded, node wait for instruction
	Map                      // Map Computation
	Collect                  // Collect Map computation results
	Reduce                   // Reduce Computation
	Convergence              // Convergence check
)

type Role int32

const (
	Master Role = iota // Master node, coordinating the network
	Worker             // Worker node, doing computation
)

type Node struct {
	Phase         Phase                      // Current computation task
	Role          Role                       // What this node has to do
	DampingFactor float64                    // C-value in PageRank algorithm
	Connection    *services.ConnectionInfo   // This node connection information
	Other         []*services.ConnectionInfo // Other nodes in the network
	// Node to contact: master for worker; client for master
	UpperLayer *services.ConnectionInfo
}
