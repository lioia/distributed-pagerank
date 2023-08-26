package pkg

import (
	"context"
	"log"
	"math"
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
			if n.Jobs == n.Responses {
				n.State.Phase = int32(Collect)
				break
			}
			if err = masterReadQueue(n); err != nil {
				break loop
			}
		case int32(Collect):
			if err = masterCollect(n); err != nil {
				break loop
			}
		case int32(Reduce):
			if n.Jobs == n.Responses {
				n.State.Phase = int32(Convergence)
				break
			}
			if err = masterReadQueue(n); err != nil {
				break loop
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
			ranks[id] = n.C*n.Data[id] + (1-n.C)*node.EValue
		}
		n.State.Phase = int32(Convergence)
		return nil
	}
	// Divide Graph in SubGraphs
	numberOfJobs := len(n.Data) / len(n.State.Others)
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
			reduce.Sums[id] = n.Data[id]
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
	n.Jobs = int32(numberOfJobs)
	n.Responses = 0
	n.Data = make(map[int32]float64)
	n.State.Phase = int32(Reduce)
	return nil
}

func masterConvergence(n *Node) error {
	convergence := 0.0
	for id, newRank := range n.Data {
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
		_, err = client.Client.SendGraph(client.Ctx, n.State.Graph)
		if err != nil {
			return err
		}
		// Node Reset
		n.State.Phase = int32(Wait)
		n.State.Graph = nil
		n.Jobs = 0
		n.Responses = 0
		n.Data = make(map[int32]float64)
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
	n.Jobs = int32(numberOfSubGraphs)
	return nil
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
	var forever chan struct{}

	// Queue Message Handler
	go func() {
		for d := range msgs {
			// Get data from bytes
			var result proto.MapIntDouble
			err := protobuf.Unmarshal(d.Body, &result)
			if err != nil {
				FailOnNack(d, err)
				continue
			}
			n.Responses += 1
			for id, v := range result.Map {
				n.Data[id] += v
			}

			// Ack
			if err := d.Ack(false); err != nil {
				FailOnNack(d, err)
				continue
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}
