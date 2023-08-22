package nodes

import (
	"log"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/pkg/services"
	"google.golang.org/protobuf/proto"
)

func (n *Node) masterUpdate() error {
	var err error
loop:
	for {
		switch n.Phase {
		case Map:
			if n.Jobs == n.Responses {
				n.Phase = Collect
				break
			}
			if err = masterReadQueue(n); err != nil {
				break loop
			}
		case Collect:
			if err = masterCollect(n); err != nil {
				break loop
			}
		case Reduce:
			if n.Jobs == n.Responses {
				n.Phase = Convergence
				break
			}
			if err = masterReadQueue(n); err != nil {
				break loop
			}
		case Convergence:
			if err = masterCollect(n); err != nil {
				break loop
			}
		}
	}
	return err
}

func masterCollect(n *Node) error {
	// TODO: publish data to queue
	n.Data = make(map[int32]float64)
	n.Phase = Reduce
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
	pkg.FailOnError("Could not register a consumer", err)
	var forever chan struct{}

	// Queue Message Handler
	go func() {
		for d := range msgs {
			// Get data from bytes
			var result services.MapIntDouble
			err := proto.Unmarshal(d.Body, &result)
			if err != nil {
				pkg.FailOnNack(d, err)
				continue
			}
			n.Responses += 1
			for id, v := range result.Map {
				n.Data[id] += v
			}

			// Ack
			if err := d.Ack(false); err != nil {
				pkg.FailOnNack(d, err)
				continue
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}
