package node

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lioia/distributed-pagerank/graph"
	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) masterUpdate() {
	status := make(chan bool)
	go masterReadQueue(n, status)
	// wait for queue registration
	<-status
	for {
		switch n.Phase {
		case Wait:
			err := masterWait(n)
			utils.FailOnError("Could not execute Wait phase", err)
		case Map:
			if n.Jobs == n.Responses {
				n.Responses = 0
				n.Phase = Collect
				utils.NodeLog("master", "Completed Map phase")
				break
			}
		case Collect:
			err := masterCollect(n)
			utils.FailOnError("Could not execute Collect phase", err)
		case Reduce:
			if n.Jobs == n.Responses {
				n.Responses = 0
				n.Phase = Convergence
				utils.NodeLog("master", "Completed Reduce phase")
				break
			}
		case Convergence:
			masterConvergence(n)
		}
		// Update every 500ms
		time.Sleep(500 * time.Millisecond)
	}
}

func masterWait(n *Node) error {
	if n.State.Graph != nil && len(n.State.Graph) > 0 {
		// A graph was loaded (from configuration or previous iteration)

		// No other node in the network -> calculating PageRank on this node
		if len(n.State.Others) == 0 {
			graph.SingleNodePageRank(n.State.Graph, n.State.C, n.State.Threshold)
			err := graph.Write(n.State.Output, n.State.Graph)
			utils.FailOnError("Failed to write graph to output %s", err, n.State.Output)
			fmt.Printf("Computation finished; output written to %s\n", n.State.Output)
			// Node Reset
			// Next time on wait, it will ask for a new graph
			n.State = &proto.State{
				Graph:     nil,
				C:         0.0,
				Threshold: 0.0,
			}
			n.Phase = Wait
			n.Jobs = 0
			n.Responses = 0
			n.Data = sync.Map{}
			utils.NodeLog("master", "Completed Wait phase on single node")
			return nil
		}
		fmt.Println("Starting computation")
		go masterSendUpdateToWorkers(n)
		err := masterWriteQueue(n, func(m map[int32]*proto.GraphNode) *proto.Job {
			mapData := make(map[int32]*proto.Map)
			dummyReduce := make(map[int32]*proto.Reduce)
			for id, v := range m {
				mapData[id] = &proto.Map{InLinks: v.InLinks}
			}
			return &proto.Job{
				Type:       0,
				MapData:    mapData,
				ReduceData: dummyReduce,
			}
		})
		if err != nil {
			return err
		}
		n.Phase = Map
		utils.NodeLog("master", "Completed Wait phase; switch to Map phase (%d jobs)", n.Jobs)
		return nil
	}
	// Ask user configuration
	fmt.Println("Start new computation")
	c := utils.ReadFloat64FromStdin("Enter c-value [in range (0.0..1.0)]: ")
	threshold := utils.ReadFloat64FromStdin("Enter threshold [in range (0.0..1.0)]: ")
	output := utils.ReadStringFromStdin("Enter output file: ")
	var g map[int32]*proto.GraphNode
	var input string
	for {
		fmt.Printf("Enter graph file [local or network resource]: ")
		fmt.Scanln(&input)
		var err error
		g, err = graph.LoadGraphResource(input)
		if err != nil {
			fmt.Println("Graph file could not be loaded correctly. Try again")
			continue
		}
		break
	}
	n.Data = sync.Map{}
	n.State.Graph = g
	n.State.C = c
	n.State.Threshold = threshold
	n.State.Output = output
	// On next Wait phase, it will send the subgraphs to the queue
	return nil
}

func masterCollect(n *Node) error {
	if len(n.State.Others) == 0 {
		// Go to wait and call single node pagerank
		n.Phase = Wait
		return nil
	}
	data := make(map[int32]float64)
	n.Data.Range(func(key, value any) bool {
		data[key.(int32)] = value.(float64)
		return true
	})
	n.Data = sync.Map{}
	err := masterWriteQueue(n, func(m map[int32]*proto.GraphNode) *proto.Job {
		dummyMap := make(map[int32]*proto.Map)
		reduce := make(map[int32]*proto.Reduce)
		for id, v := range m {
			reduce[id] = &proto.Reduce{
				Sum: data[id],
				E:   v.E,
			}
		}
		return &proto.Job{
			Type:       1,
			ReduceData: reduce,
			MapData:    dummyMap,
		}
	})
	if err != nil {
		return err
	}
	// Switch to Reduce phase
	n.Phase = Reduce
	utils.NodeLog("master", "Completed Collect phase; switch to Reduce phase (%d jobs)", n.Jobs)
	return nil
}

func masterConvergence(n *Node) {
	var convergence float64
	n.Data.Range(func(key, value any) bool {
		id := key.(int32)
		newRank := value.(float64)
		oldRank := n.State.Graph[id].Rank
		convergence += math.Abs(newRank - oldRank)
		// After calculating the convergence value, it can be safely updated
		n.State.Graph[id].Rank = newRank
		return true
	})
	for _, u := range n.State.Graph {
		for j, v := range u.InLinks {
			v.Rank = n.State.Graph[j].Rank
		}
	}
	if convergence > n.State.Threshold {
		// Does not converge -> iterate
		utils.NodeLog("master", "Convergence check failed (%f)", convergence)
		// Start new computation with updated pagerank values
		n.Phase = Wait
	} else {
		utils.NodeLog("master", "Convergence check success (%f)", convergence)
		// Normalize values
		rankSum := 0.0
		for _, node := range n.State.Graph {
			rankSum += node.Rank
		}
		for id := range n.State.Graph {
			n.State.Graph[id].Rank /= rankSum
		}
		err := graph.Write(n.State.Output, n.State.Graph)
		utils.FailOnError("Failed to write graph to output %s", err, n.State.Output)
		fmt.Printf("Computation finished; output written to %s\n", n.State.Output)
		// Node Reset
		n.State = &proto.State{
			Graph:     nil,
			C:         0.0,
			Threshold: 0.0,
		}
		n.Phase = Wait
		n.Jobs = 0
		n.Responses = 0
	}
	n.Data = sync.Map{}
}

// Master send state to all workers
func masterSendUpdateToWorkers(n *Node) {
	crashedWorkers := make(map[int]bool)
	for i, v := range n.State.Others {
		worker, err := utils.NodeCall(v)
		if err != nil {
			utils.ServerLog("[WARN] Worker %s crashed", v)
			crashedWorkers[i] = true
			continue
		}
		defer worker.Close()
		_, err = worker.Client.StateUpdate(worker.Ctx, n.State)
		if err != nil {
			utils.ServerLog("[WARN] Worker %s crashed", v)
			crashedWorkers[i] = true
		}
	}
	// Remove crashed
	var newWorkers []string
	for i, v := range n.State.Others {
		if !crashedWorkers[i] {
			newWorkers = append(newWorkers, v)
		}
	}
	n.State.Others = newWorkers
}

// Master send other state update to workers
func masterSendOtherStateUpdate(n *Node) {
	crashedWorkers := make(map[int]bool)
	for i, v := range n.State.Others {
		worker, err := utils.NodeCall(v)
		if err != nil {
			crashedWorkers[i] = true
			continue
		}
		defer worker.Close()
		others := proto.OtherState{Connections: n.State.Others}
		_, err = worker.Client.OtherStateUpdate(worker.Ctx, &others)
		if err != nil {
			crashedWorkers[i] = true
		}
	}

	// Remove crashed
	var newWorkers []string
	for i, v := range n.State.Others {
		if !crashedWorkers[i] {
			newWorkers = append(newWorkers, v)
		}
	}
	n.State.Others = newWorkers

	// Do not send state update to working workers
	// It will happen on next state update
}

func masterWriteQueue(n *Node, fn func(map[int32]*proto.GraphNode) *proto.Job) error {
	// Divide Graph in SubGraphs
	numberOfJobs := len(n.State.Others)
	if numberOfJobs >= len(n.State.Graph) {
		numberOfJobs = len(n.State.Graph)
	}
	subGraphs := make([]map[int32]*proto.GraphNode, numberOfJobs)
	i := 0
	for id, node := range n.State.Graph {
		if subGraphs[i] == nil {
			subGraphs[i] = make(map[int32]*proto.GraphNode)
		}
		subGraphs[i][id] = node
		i = (i + 1) % numberOfJobs
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		job := fn(subGraph)
		data, err := protobuf.Marshal(job)
		if err != nil {
			return err
		}
		err = n.Queue.Channel.PublishWithContext(ctx,
			"",
			n.Queue.Work.Name, // routing key
			false,             // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/x-protobuf",
				Body:         data,
			})
		if err != nil {
			return err
		}
	}
	n.Jobs = numberOfJobs
	return nil
}

func masterReadQueue(n *Node, status chan bool) {
	// Register consumer
	msgs, err := n.Queue.Channel.Consume(
		n.Queue.Result.Name, // queue
		"",                  // consumer
		false,               // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	utils.FailOnError("Could not register a consumer for %s queue", err, n.Queue.Result.Name)
	utils.NodeLog("master", "Registered consumer for queue %s", n.Queue.Result.Name)
	status <- true
	for msg := range msgs {
		var result proto.Result
		err := protobuf.Unmarshal(msg.Body, &result)
		if err != nil {
			utils.FailOnNack(msg, err)
			continue
		}
		for id, v := range result.Values {
			oldValue, ok := n.Data.Load(id)
			newValue := v
			if ok {
				newValue += oldValue.(float64)
			}
			n.Data.Store(id, newValue)
		}

		// Ack
		if err := msg.Ack(false); err != nil {
			utils.FailOnNack(msg, err)
			continue
		}
		// Correctly read message
		n.Responses += 1
	}
}
