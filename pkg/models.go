package pkg

import (
	"github.com/lioia/distributed-pagerank/proto"
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
	State      *proto.State // Shared node state
	Role       Role         // What this node has to do
	C          float64      // C-value in PageRank algorithm
	Threshold  float64      // Threshold value used in PageRank algorithm
	Connection string       // This node connection information
	Queue      Queue        // Queue information
	Master     string       // Master node (set if this node is a worker)
	Candidacy  int64        // Timestamp of new candidacy (0: no candidate)
}

type Queue struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Work    *amqp.Queue
	Result  *amqp.Queue
}

func RoleToString(role Role) string {
	switch role {
	case Master:
		return "Master"
	case Worker:
		return "Worker"
	}
	return "Undefined"
}

func (n *Node) Update() {
	if n.Role == Worker {
		n.workerUpdate()
	} else if n.Role == Master {
		n.masterUpdate()
	}
}
