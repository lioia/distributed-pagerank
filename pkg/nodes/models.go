package nodes

import (
	"github.com/lioia/distributed-pagerank/pkg/services"
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
	Phase      Phase             // Current computation task
	Role       Role              // What this node has to do
	C          float64           // C-value in PageRank algorithm
	Threshold  float64           // Threshold value used in PageRank algorithm
	Connection string            // This node connection information
	Others     []string          // Other nodes in the network
	Queue      Queue             // Queue information
	UpperLayer string            // Node to contact (worker -> master; master -> client)
	Graph      *services.Graph   // Graph Structure
	Jobs       int32             // Number of jobs dispatched to worker nodes
	Responses  int32             // Number of responses received
	Data       map[int32]float64 // Intermediate data received from map/reduce
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

func (n *Node) Update() error {
	if n.Role == Worker {
		return n.workerUpdate()
	} else if n.Role == Master {
		return n.masterUpdate()
	}
	return nil
}
