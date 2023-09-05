package node

import (
	"fmt"

	"github.com/lioia/distributed-pagerank/graph"
	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"
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
	State       *proto.State // Shared node state
	Role        Role         // What this node has to do
	Connection  string       // This node connection information
	Queue       Queue        // Queue information
	Master      string       // Master node (set if this node is a worker)
	Candidacy   int64        // Timestamp of new candidacy (0: no candidate)
	QueueReader chan bool    // Cancel channel for worker goroutine
	Responses   int32        // Master state: number of read result messages
	// Master state: Data collected from result queue
	// Standard map is not thread safe
	// NOTE: look into sync.Map
	Data *utils.SafeMap[int32, float64]
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

func (n *Node) InitializeMaster() error {
	config, err := utils.LoadConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to read configuration file: %v", err)
	}
	graph, err := graph.LoadGraphResource(config.Graph)
	if err != nil {
		return fmt.Errorf("Failed to read graph: %v", err)
	}
	n.State.C = config.C
	n.State.Threshold = config.Threshold
	n.State.Graph = graph
	return nil
}

func (n *Node) InitializeWorker(master string, join *proto.Join) (workQueue, resultQueue string) {
	n.Role = Worker
	n.Master = master
	n.State = join.State
	workQueue = join.WorkQueue
	resultQueue = join.ResultQueue
	n.QueueReader = make(chan bool)
	return
}

func (n *Node) Update() {
	if n.Role == Worker {
		n.workerUpdate()
	} else if n.Role == Master {
		n.masterUpdate()
	}
}
