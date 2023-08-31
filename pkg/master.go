package pkg

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"github.com/lioia/distributed-pagerank/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) masterUpdate() {
	status := make(chan bool)
	go masterReadQueue(n, status)
	for {
		switch n.State.Phase {
		case int32(Map):
			numberOfMessages := GetNumberOfQueueMessages(n.Queue.Result.Name)
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
			FailOnError("Could not execute Collect phase", err)
			log.Println("Completed Collect phase")
			log.Printf("Switch to Reduce phase (%d jobs)\n", n.State.Jobs)
		case int32(Reduce):
			numberOfMessages := GetNumberOfQueueMessages(n.Queue.Result.Name)
			if n.State.Jobs == int32(numberOfMessages) {
				status <- true
				<-status
				n.State.Phase = int32(Convergence)
				go n.masterSendUpdateToWorkers()
				log.Println("Completed Reduce phase")
				break
			}
		case int32(Convergence):
			err := masterConvergence(n)
			FailOnError("Could not execute Convergence phase", err)
			log.Println("Completed Convergence phase")
		}
		// Update every 500ms
		time.Sleep(500 * time.Millisecond)
	}
}

func masterCollect(n *Node) error {
	if len(n.State.Others) == 0 {
		ranks := make(map[int32]float64)
		for id, node := range n.State.Graph.Graph {
			ranks[id] = n.C*n.State.Data[id] + (1-n.C)*node.EValue
		}
		n.State.Phase = int32(Convergence)
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfJobs := len(n.State.Others)
	if numberOfJobs >= len(n.State.Graph.Graph) {
		numberOfJobs = len(n.State.Graph.Graph) - 1
	}
	subGraphs := make([]*proto.Graph, numberOfJobs)
	i := 0
	for id, node := range n.State.Graph.Graph {
		if subGraphs[i] == nil {
			subGraphs[i] = &proto.Graph{Graph: make(map[int32]*proto.GraphNode)}
		}
		subGraphs[i].Graph[id] = node
		i = (i + 1) % numberOfJobs
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		reduce := proto.Reduce{Sums: make(map[int32]float64)}
		for id, v := range subGraph.Graph {
			reduce.Nodes = append(reduce.Nodes, v)
			reduce.Sums[id] = n.State.Data[id]
		}
		job := proto.Job{Type: 1, ReduceData: &reduce, MapData: subGraph.Graph}
		data, err := protobuf.Marshal(&job)
		if err != nil {
			EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
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
			EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
			return err
		}
	}
	// Switch to Reduce phase
	n.State.Jobs = int32(numberOfJobs)
	n.State.Phase = int32(Reduce)
	return nil
}

func masterConvergence(n *Node) error {
	convergence := 0.0
	for id, newRank := range n.State.Data {
		oldRank := n.State.Graph.Graph[id].Rank
		convergence += math.Abs(newRank - oldRank)
		// After calculating the convergence value, it can be safely updated
		n.State.Graph.Graph[id].Rank = newRank
	}
	// Does not converge -> iterate
	if convergence > n.Threshold {
		// Start new computation with updated pagerank values
		return n.WriteGraphToQueue()
	} else {
		log.Println("Convergence check success; sending results to client")
		// Send results to client
		// TODO: Crashes because the container cannot contact the host
		client, err := ApiCall(n.State.Client)
		if err != nil {
			return err
		}
		_, err = client.Client.ReceiveResults(client.Ctx, n.State.Graph)
		if err != nil {
			return err
		}
		// Node Reset
		n.State = &proto.State{Phase: int32(Wait)}
		go n.masterSendUpdateToWorkers()
	}

	return nil
}

func (n *Node) WriteGraphToQueue() error {
	if n.Role == Worker {
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfSubGraphs := len(n.State.Others)
	if numberOfSubGraphs >= len(n.State.Graph.Graph) {
		numberOfSubGraphs = len(n.State.Graph.Graph) - 1
	}
	subGraphs := make([]*proto.Graph, numberOfSubGraphs)
	i := 0
	for id, node := range n.State.Graph.Graph {
		if subGraphs[i] == nil {
			subGraphs[i] = &proto.Graph{Graph: make(map[int32]*proto.GraphNode)}
		}
		subGraphs[i].Graph[id] = node
		i = (i + 1) % numberOfSubGraphs
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		job := proto.Job{Type: 0, MapData: subGraph.Graph, ReduceData: &proto.Reduce{}}
		data, err := protobuf.Marshal(&job)
		if err != nil {
			EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
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
			EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
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
func (n *Node) masterSendUpdateToWorkers() {
	var wg sync.WaitGroup
	crashed := make(chan int)
	for i, v := range n.State.Others {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, url string, crashed chan int) {
			worker, err := NodeCall(url)
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
	FailOnError("Could not register a consumer", err)
	for {
		// Wait for status update
		<-status
		n.State.Data = make(map[int32]float64)

		for i := 0; i < int(n.State.Jobs); i++ {
			msg := <-msgs
			var result proto.MapIntDouble
			err := protobuf.Unmarshal(msg.Body, &result)
			// FIXME: better error handling
			if err != nil {
				FailOnNack(msg, err)
				continue
			}
			for id, v := range result.Map {
				n.State.Data[id] += v
			}

			// Ack
			// FIXME: better error handling
			if err := msg.Ack(false); err != nil {
				FailOnNack(msg, err)
				continue
			}
		}
		// Correctly read every message
		status <- true
	}
}
