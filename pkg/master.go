package pkg

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lioia/distributed-pagerank/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) masterUpdate() error {
	var err error
loop:
	for {
		switch n.State.Phase {
		case int32(Map):
			if n.State.Jobs == int32(n.Queue.Work.Messages) {
				if err = masterReadQueue(n); err != nil {
					break loop
				}
				n.masterSendUpdateToWorkers()
				n.State.Phase = int32(Collect)
				break
			}
		case int32(Collect):
			if err = masterCollect(n); err != nil {
				break loop
			}
		case int32(Reduce):
			if n.State.Jobs == int32(n.Queue.Work.Messages) {
				if err = masterReadQueue(n); err != nil {
					break loop
				}
				n.masterSendUpdateToWorkers()
				n.State.Phase = int32(Convergence)
				break
			}
		case int32(Convergence):
			if err = masterConvergence(n); err != nil {
				break loop
			}
		}
	}
	return err
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
	numberOfJobs := len(n.State.Data) / len(n.State.Others)
	subGraphs := make([]*proto.Graph, numberOfJobs)
	index := 0
	for id, node := range n.State.Graph.Graph {
		subGraphs[index/numberOfJobs].Graph[id] = node
		index += 1
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		reduce := proto.Reduce{}
		for id, v := range subGraph.Graph {
			reduce.Nodes = append(reduce.Nodes, v)
			reduce.Sums[id] = n.State.Data[id]
		}
		job := proto.Job{Type: 1, ReduceData: &reduce}
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
	n.State.Data = make(map[int32]float64)
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
		// Send results to client
		client, err := ClientCall(n.State.Client)
		if err != nil {
			return err
		}
		_, err = client.Client.ReceiveResults(client.Ctx, n.State.Graph)
		if err != nil {
			return err
		}
		// Node Reset
		n.State = &proto.State{Phase: int32(Wait)}
		n.masterSendUpdateToWorkers()
	}

	return nil
}

func (n *Node) WriteGraphToQueue() error {
	if n.Role == Worker {
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfSubGraphs := len(n.State.Graph.Graph) / len(n.State.Others)
	subGraphs := make([]*proto.Graph, numberOfSubGraphs)
	index := 0
	for id, node := range n.State.Graph.Graph {
		subGraphs[index/numberOfSubGraphs].Graph[id] = node
		index += 1
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		job := proto.Job{Type: 0, MapData: subGraph.Graph}
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

func masterReadQueue(n *Node) error {
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

	for msg := range msgs {
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

	return nil
}
