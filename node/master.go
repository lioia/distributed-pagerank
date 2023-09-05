package node

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/lioia/distributed-pagerank/graph"
	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) masterUpdate() {
	go masterReadQueue(n)
	for {
		switch n.State.Phase {
		case int32(Wait):
			err := masterWait(n)
			utils.FailOnError("Could not execute Wait phase", err)
		case int32(Map):
			if n.State.Jobs == n.Responses {
				n.Responses = 0
				n.State.Phase = int32(Collect)
				log.Println("Completed Map phase")
				break
			}
		case int32(Collect):
			err := masterCollect(n)
			utils.FailOnError("Could not execute Collect phase", err)
			log.Println("Completed Collect phase")
			log.Printf("Switch to Reduce phase (%d jobs)\n", n.State.Jobs)
		case int32(Reduce):
			if n.State.Jobs == n.Responses {
				n.Responses = 0
				n.State.Phase = int32(Convergence)
				log.Println("Completed Reduce phase")
				break
			}
		case int32(Convergence):
			masterConvergence(n)
			log.Println("Completed Convergence phase")
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
			fmt.Println("Computed Page Rank on single node")
			for id, v := range n.State.Graph {
				fmt.Printf("%d -> %f\n", id, v.Rank)
			}
			// Node Reset
			// Next time on wait, it will ask for a new graph
			n.State = &proto.State{
				Graph:     nil,
				C:         0.0,
				Threshold: 0.0,
				Jobs:      0,
				Phase:     int32(Wait),
			}
			n.Data = utils.NewSafeMap[int32, float64]()
			log.Println("Completed Wait phase")
			return nil
		}
		go n.masterSendUpdateToWorkers()
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
		log.Println("Completed Wait phase")
		log.Printf("Switch to Map phase (%d jobs)\n", n.State.Jobs)
		n.State.Phase = int32(Map)
		return nil
	}
	// Ask user configuration
	fmt.Println("Start new computation:")
	c := utils.ReadFloat64FromStdin("Enter c-value [in range (0.0..1.0)] ")
	threshold := utils.ReadFloat64FromStdin("Enter threshold [in range (0.0..1.0)] ")
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
	n.Data = utils.NewSafeMap[int32, float64]()
	n.State.Graph = g
	n.State.C = c
	n.State.Threshold = threshold
	// On next Wait phase, it will send the subgraphs to the queue
	return nil
}

func masterCollect(n *Node) error {
	if len(n.State.Others) == 0 {
		ranks := make(map[int32]float64)
		for id, node := range n.State.Graph {
			ranks[id] = n.State.C*n.Data.Get(id) + (1-n.State.C)*node.E
		}
		n.State.Phase = int32(Convergence)
		return nil
	}
	go n.masterSendUpdateToWorkers()
	data := n.Data.Clone()
	n.Data.Reset()
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
	n.State.Phase = int32(Reduce)
	return nil
}

func masterConvergence(n *Node) {
	var convergence float64
	for _, id := range n.Data.Keys() {
		newRank := n.Data.Get(id)
		oldRank := n.State.Graph[id].Rank
		convergence += math.Abs(newRank - oldRank)
		// After calculating the convergence value, it can be safely updated
		n.State.Graph[id].Rank = newRank
	}
	// Update InLinks rank values
	for _, u := range n.State.Graph {
		for j, v := range u.InLinks {
			v.Rank = n.State.Graph[j].Rank
		}
	}
	if convergence > n.State.Threshold {
		// Does not converge -> iterate
		log.Printf("Convergence check failed (%f)", convergence)
		// Start new computation with updated pagerank values
		n.State.Phase = int32(Wait)
	} else {
		log.Println("Convergence check success")
		// Normalize values
		rankSum := 0.0
		for _, node := range n.State.Graph {
			rankSum += node.Rank
		}
		for id, v := range n.State.Graph {
			n.State.Graph[id].Rank /= rankSum
			fmt.Printf("%d -> %f\n", id, v.Rank)
		}
		// Node Reset
		n.State = &proto.State{
			Graph:     nil,
			C:         0.0,
			Threshold: 0.0,
			Jobs:      0,
			Phase:     int32(Wait),
		}
	}
	n.Data = utils.NewSafeMap[int32, float64]()
}

// Master send state to all workers
func (n *Node) masterSendUpdateToWorkers() {
	crashed := make(chan int)
	for i, v := range n.State.Others {
		worker, err := utils.NodeCall(v)
		if err != nil {
			crashed <- i
		}
		defer worker.Close()
		_, err = worker.Client.StateUpdate(worker.Ctx, n.State)
		if err != nil {
			log.Printf("Worker %s crashed", v)
			crashed <- i
		}
	}

	// Close to no cause any leaks
	close(crashed)

	// Collect crashed
	crashedWorkers := make(map[int]bool)
	for i := range crashed {
		crashedWorkers[i] = true
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
	n.State.Jobs = int32(numberOfJobs)
	return nil
}

func masterReadQueue(n *Node) {
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
	utils.FailOnError("Could not register a consumer", err)
	log.Println("Master registered consumer")
	for msg := range msgs {
		var result proto.Result
		err := protobuf.Unmarshal(msg.Body, &result)
		// FIXME: better error handling
		if err != nil {
			utils.FailOnNack(msg, err)
			continue
		}
		for id, v := range result.Values {
			n.Data.Increment(id, v)
		}

		// Ack
		// FIXME: better error handling
		if err := msg.Ack(false); err != nil {
			utils.FailOnNack(msg, err)
			continue
		}
		// Correctly read message
		n.Responses += 1
	}
}
