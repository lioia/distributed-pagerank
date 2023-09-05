package node

import (
	"context"
	"log"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"
	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
)

func (n *Node) workerUpdate() {
	go func() {
		select {
		case <-n.QueueReader:
			log.Println("Queue Reading goroutine canceled")
			return
		default:
			for {
				readQueue(n)
			}
		}
	}()
	// Worker Health Check
	healthCheck := utils.ReadIntEnvVarOr("HEALTH_CHECK", 5000)
	for {
		n.WorkerHealthCheck()
		time.Sleep(time.Duration(healthCheck) * time.Millisecond)
	}
}

func readQueue(n *Node) {
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
	utils.FailOnError("Could not register a consumer", err)
	log.Println("Worker registered consumer")
	// Queue Message Handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for d := range msgs {
		// Get data from bytes
		var job proto.Job
		err := protobuf.Unmarshal(d.Body, &job)
		if err != nil {
			utils.FailOnNack(d, err)
			continue
		}
		result := proto.Result{}
		// Create result value
		// Handle job based on type
		if job.Type == 0 {
			log.Println("Computing Map Job")
			result.Values = n.workerMap(job.MapData)
			log.Println("Completed Map Job")
		} else if job.Type == 1 {
			log.Println("Computing Reduce Job")
			result.Values = n.workerReduce(job.ReduceData)
			log.Println("Completed Reduce Job")
		}
		// Publish result to Result queue
		data, err := protobuf.Marshal(&result)
		if err != nil {
			utils.FailOnNack(d, err)
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
			utils.FailOnNack(d, err)
			continue
		}

		// Ack
		if err := d.Ack(false); err != nil {
			utils.FailOnNack(d, err)
			continue
		}
	}
}

func (n *Node) workerMap(subGraph map[int32]*proto.Map) map[int32]float64 {
	contributions := make(map[int32]float64)
	for id, u := range subGraph {
		for _, v := range u.InLinks {
			contributions[id] += v.Rank / float64(v.Outlinks)
		}
	}
	return contributions
}

func (n *Node) workerReduce(reduce map[int32]*proto.Reduce) map[int32]float64 {
	ranks := make(map[int32]float64)
	for id, v := range reduce {
		ranks[id] = n.State.C*v.Sum + (1-n.State.C)*v.E
	}
	return ranks
}

func (n *Node) WorkerHealthCheck() {
	master, err := utils.NodeCall(n.Master)
	if err != nil {
		// Master didn't respond -> assuming crash
		n.workerCandidacy()
		return
	}
	defer master.Conn.Close()
	defer master.CancelFunc()
	health, err := master.Client.HealthCheck(master.Ctx, nil)
	if err != nil {
		// Master didn't respond -> assuming crash
		n.workerCandidacy()
		return
	}
	// No error detected -> master is still valid
	if state := health.GetState(); state != nil {
		// Last state update was missed -> loading now
		n.State = state
	}
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
		worker, err := utils.NodeCall(v)
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
	if elected {
		// Stop goroutines
		n.QueueReader <- true
		// Switch to master
		n.Role = Master
		// Empty queues
		utils.EmptyQueue(n.Queue.Channel, n.Queue.Work.Name)
		utils.EmptyQueue(n.Queue.Channel, n.Queue.Result.Name)
		// Start master update
		n.Update()
	}
}
