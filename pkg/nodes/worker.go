package nodes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/pkg/services"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

func (n *Node) workerUpdate() error {
	// Register consumer
	msgs, err := n.Queue.Channel.Consume(
		n.Queue.Work.Name, // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	pkg.FailOnError("Could not register a consumer", err)
	var forever chan struct{}

	// Queue Message Handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for d := range msgs {
			// Get data from bytes
			var job services.Job
			err := proto.Unmarshal(d.Body, &job)
			if err != nil {
				FailOnNack(d, err)
				continue
			}
			// Create result value
			result := services.Result{Type: job.Type}
			// Handle job based on type
			if job.Type == 0 {
				result.Data = n.workerMap(job.MapData)
			} else if job.Type == 1 {
				result.Data = n.workerReduce(job.ReduceData)
			}
			// Publish result to Result queue
			data, err := proto.Marshal(&result)
			if err != nil {
				FailOnNack(d, err)
				continue
			}
			err = n.Queue.Channel.PublishWithContext(ctx,
				"",
				n.Queue.Result.Name, // routing key
				false,               // mandatory
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  "application/x-protobuf",
					Body:         data,
				})
			if err != nil {
				FailOnNack(d, err)
				continue
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

func (n *Node) workerMap(subGraph map[int32]*services.GraphNode) map[int32]float64 {
	contributions := make(map[int32]float64)
	for _, u := range subGraph {
		data := pkg.ComputeMap(u)
		for id, v := range data {
			contributions[id] += v
		}
	}
	return contributions
}

func (n *Node) workerReduce(data *services.Reduce) map[int32]float64 {
	ranks := make(map[int32]float64)
	for _, v := range data.Node {
		ranks[v.Id] = pkg.ComputeReduce(v, data.Sums[v.Id], n.C)
	}
	return ranks
}

func FailOnNack(d amqp.Delivery, err error) {
	fmt.Printf("Could not marshal result: %v", err)
	// Message will be re-added to the queue
	if err = d.Nack(false, true); err != nil {
		log.Fatalf("Could not NACK to message queue: %v", err)
	}
}
