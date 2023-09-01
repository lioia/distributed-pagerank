package node

import (
	"context"
	"fmt"
	"log"
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
	for {
		switch n.State.Phase {
		case int32(Wait):
			err := masterWait(n)
			utils.FailOnError("Could not execute Wait phase", err)
			log.Println("Completed Wait phase")
		case int32(Map):
			numberOfMessages := utils.GetNumberOfQueueMessages(n.Queue.Result.Name)
			if n.State.Jobs == int32(numberOfMessages) {
				status <- true
				<-status
				n.State.Phase = int32(Collect)
				go n.masterSendUpdateToWorkers()
				log.Println("Completed Map phase")
				break
			}
		case int32(Collect):
			err := masterCollect(n)
			utils.FailOnError("Could not execute Collect phase", err)
			log.Println("Completed Collect phase")
			log.Printf("Switch to Reduce phase (%d jobs)\n", n.State.Jobs)
		case int32(Reduce):
			numberOfMessages := utils.GetNumberOfQueueMessages(n.Queue.Result.Name)
			if n.State.Jobs == int32(numberOfMessages) {
				status <- true
				<-status
				n.State.Phase = int32(Convergence)
				go n.masterSendUpdateToWorkers()
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
	if n.State.Graph != nil {
		// A graph was loaded (from configuration or previous iteration)

		// No other node in the network -> calculating PageRank on this node
		if len(n.State.Others) == 0 {
			graph.SingleNodePageRank(n.State.Graph, n.State.C, n.State.Threshold)
			log.Println("Computed Page Rank on single node")
			for id, v := range n.State.Graph {
				log.Printf("%d -> %f\n", id, v.Rank)
			}
			// Node Reset
			n.State.Graph = nil
			n.State.C = 0.0
			n.State.Threshold = 0.0
			return nil
		}
		// Divide the graph in message and publish to queue
		err := n.WriteGraphToQueue()
		if err != nil {
			return err
		}
		n.State.Phase = int32(Map)
		// Send state update to worker nodes
		go n.masterSendUpdateToWorkers()
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
			ranks[id] = n.State.C*n.State.Data[id] + (1-n.State.C)*node.EValue
		}
		n.State.Phase = int32(Convergence)
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfJobs := len(n.State.Others)
	if numberOfJobs >= len(n.State.Graph) {
		numberOfJobs = len(n.State.Graph) - 1
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
		dummyMap := make(map[int32]*proto.Map)
		reduce := make(map[int32]*proto.Reduce)
		for id, v := range subGraph {
			reduce[id] = &proto.Reduce{
				Sum:    n.State.Data[id],
				EValue: v.EValue,
			}
		}
		job := proto.Job{Type: 1, ReduceData: reduce, MapData: dummyMap}
		data, err := protobuf.Marshal(&job)
		if err != nil {
			utils.EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
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
			utils.EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
			return err
		}
	}
	// Switch to Reduce phase
	n.State.Jobs = int32(numberOfJobs)
	n.State.Phase = int32(Reduce)
	return nil
}

func masterConvergence(n *Node) {
	convergence := 0.0
	for id, newRank := range n.State.Data {
		oldRank := n.State.Graph[id].Rank
		convergence += math.Abs(newRank - oldRank)
		// After calculating the convergence value, it can be safely updated
		n.State.Graph[id].Rank = newRank
	}
	// Does not converge -> iterate
	if convergence > n.State.Threshold {
		// Start new computation with updated pagerank values
		n.State.Phase = int32(Wait)
	} else {
		log.Println("Convergence check success")
		for id, v := range n.State.Graph {
			log.Printf("%d -> %f\n", id, v.Rank)
		}
		// Node Reset
		n.State = &proto.State{
			Graph:     nil,
			C:         0.0,
			Threshold: 0.0,
			Jobs:      0,
			Phase:     int32(Wait),
		}
		go n.masterSendUpdateToWorkers()
	}
}

func (n *Node) WriteGraphToQueue() error {
	if n.Role == Worker {
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfSubGraphs := len(n.State.Others)
	if numberOfSubGraphs >= len(n.State.Graph) {
		numberOfSubGraphs = len(n.State.Graph) - 1
	}
	subGraphs := make([]map[int32]*proto.GraphNode, numberOfSubGraphs)
	i := 0
	for id, node := range n.State.Graph {
		if subGraphs[i] == nil {
			subGraphs[i] = make(map[int32]*proto.GraphNode)
		}
		subGraphs[i][id] = node
		i = (i + 1) % numberOfSubGraphs
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		mapData := make(map[int32]*proto.Map)
		dummyReduce := make(map[int32]*proto.Reduce)
		for id, v := range subGraph {
			mapData[id] = &proto.Map{
				Rank:     v.Rank,
				OutLinks: v.OutLinks,
			}
		}
		job := proto.Job{Type: 0, MapData: mapData, ReduceData: dummyReduce}
		data, err := protobuf.Marshal(&job)
		if err != nil {
			utils.EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
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
			utils.EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
			return err
		}
	}
	// Switch to Map phase
	n.State.Phase = int32(Map)
	n.State.Jobs = int32(numberOfSubGraphs)
	log.Printf("Switch to Map phase (%d jobs)\n", n.State.Jobs)
	return nil
}

// Master send state to all workers
// TODO: maybe goroutine is not necessary (since we're already in one)
func (n *Node) masterSendUpdateToWorkers() {
	var wg sync.WaitGroup
	crashed := make(chan int)
	for i, v := range n.State.Others {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, url string, crashed chan int) {
			worker, err := utils.NodeCall(url)
			if err != nil {
				crashed <- i
			}
			_, err = worker.Client.StateUpdate(worker.Ctx, n.State)
			if err != nil {
				crashed <- i
			}
		}(&wg, i, v, crashed)
	}

	// Wait for all goroutines to finish
	wg.Done()
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

func masterReadQueue(n *Node, status chan bool) {
	log.Println("Master registered consumer")
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
	for {
		// Wait for status update
		<-status
		n.State.Data = make(map[int32]float64)

		for i := 0; i < int(n.State.Jobs); i++ {
			msg := <-msgs
			var result proto.Result
			err := protobuf.Unmarshal(msg.Body, &result)
			// FIXME: better error handling
			if err != nil {
				utils.FailOnNack(msg, err)
				continue
			}
			for id, v := range result.Values {
				n.State.Data[id] += v
			}

			// Ack
			// FIXME: better error handling
			if err := msg.Ack(false); err != nil {
				utils.FailOnNack(msg, err)
				continue
			}
		}
		// Correctly read every message
		status <- true
	}
}
