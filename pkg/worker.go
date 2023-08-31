package pkg

import (
	"context"
	"log"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) workerUpdate() {
	healthCheck := ReadIntEnvVarOr("HEALTH_CHECK", 5000)
	// Worker Health Check
	go func() {
		for {
			n.WorkerHealthCheck()
			time.Sleep(time.Duration(healthCheck) * time.Millisecond)
		}
	}()
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
	FailOnError("Could not register a consumer", err)
	var forever chan struct{}

	// Queue Message Handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for d := range msgs {
			// Get data from bytes
			var job proto.Job
			err := protobuf.Unmarshal(d.Body, &job)
			if err != nil {
				FailOnNack(d, err)
				continue
			}
			result := proto.MapIntDouble{}
			// Create result value
			// Handle job based on type
			if job.Type == 0 {
				log.Println("Computing Map Job")
				result.Map = n.workerMap(job.MapData)
				log.Println("Completed Map Job")
			} else if job.Type == 1 {
				log.Println("Computing Reduce Job")
				result.Map = n.workerReduce(job.ReduceData)
				log.Println("Completed Reduce Job")
			}
			// Publish result to Result queue
			data, err := protobuf.Marshal(&result)
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
}

func (n *Node) workerMap(subGraph map[int32]*proto.GraphNode) map[int32]float64 {
	contributions := make(map[int32]float64)
	for _, u := range subGraph {
		data := ComputeMap(u)
		for id, v := range data {
			contributions[id] += v
		}
	}
	return contributions
}

func (n *Node) workerReduce(data *proto.Reduce) map[int32]float64 {
	ranks := make(map[int32]float64)
	for _, v := range data.Nodes {
		ranks[v.Id] = ComputeReduce(v, data.Sums[v.Id], n.C)
	}
	return ranks
}

func (n *Node) WorkerHealthCheck() {
	master, err := NodeCall(n.Master)
	if err != nil {
		// Master didn't respond -> assuming crash
		n.workerCandidacy()
		return
	}
	defer master.Conn.Close()
	defer master.CancelFunc()
	_, err = master.Client.HealthCheck(master.Ctx, nil)
	if err != nil {
		// Master didn't respond -> assuming crash
		n.workerCandidacy()
		return
	}
	// No error detected -> master is still valid
}

func (n *Node) workerCandidacy() {
	candidacy := &proto.Candidacy{
		Connection: n.Connection,
		Timestamp:  time.Now().UnixMilli(),
	}
	n.Candidacy = candidacy.Timestamp
	crashedWorkers := make(map[int]bool)
	elected := true
	for i, v := range n.State.Others {
		worker, err := NodeCall(v)
		if err != nil {
			crashedWorkers[i] = true
			continue
		}
		defer worker.Conn.Close()
		defer worker.CancelFunc()
		ack, err := worker.Client.MasterCandidate(worker.Ctx, candidacy)
		if err != nil {
			crashedWorkers[i] = true
			continue
		}
		// NACK -> there is an older candidacy
		if !ack.Ack {
			n.Master = ack.Candidate
			elected = false
			break
		}
	}
	// Remove crashed workers from state
	var newWorkers []string
	for i, v := range n.State.Others {
		if !crashedWorkers[i] {
			newWorkers = append(newWorkers, v)
		}
	}
	n.State.Others = newWorkers
	// TODO: start API server
	if elected {
		n.Role = Master
	}
}
