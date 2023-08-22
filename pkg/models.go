package pkg

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

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
	Phase      Phase    // Current computation task
	Role       Role     // What this node has to do
	C          float64  // C-value in PageRank algorithm
	Connection string   // This node connection information
	Other      []string // Other nodes in the network
	Queue      Queue    // Queue information
	UpperLayer string   // Node to contact (worker -> master; master -> client)
}

type Queue struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Work    *amqp.Queue
	Result  *amqp.Queue
}
