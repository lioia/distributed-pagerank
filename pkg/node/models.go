package node

import (
	"sync"

	"github.com/lioia/distributed-pagerank/pkg/utils"
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
	State       *proto.State // Shared node state
	Role        Role         // What this node has to do
	Connection  string       // This node connection information
	Queue       Queue        // Queue information
	Master      string       // Master node (set if this node is a worker)
	Candidacy   int64        // Timestamp of new candidacy (0: no candidate)
	QueueReader chan bool    // Cancel channel for worker goroutine
	Phase       Phase        // Master state: current computation (master as a FSM)
	Jobs        int          // Master state: number of jobs in the work queue
	Responses   int          // Master state: number of read result messages
	Data        sync.Map     // Master state: data collected from result queue (std map is no thread safe)
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
		utils.NodeLog("worker", "update")
		n.workerUpdate()
	} else if n.Role == Master {
		utils.NodeLog("master", "update")
		n.masterUpdate()
	}
}
