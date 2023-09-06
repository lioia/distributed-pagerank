package node

import (
	"context"
	"time"

	"github.com/lioia/distributed-pagerank/proto"
	"github.com/lioia/distributed-pagerank/utils"
	amqp "github.com/rabbitmq/amqp091-go"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (n *Node) workerUpdate() {
	go readQueue(n)
	healthCheck := utils.ReadIntEnvVarOr("HEALTH_CHECK", 1000)
	for {
		time.Sleep(time.Duration(healthCheck) * time.Millisecond)
		workerHealthCheck(n)
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
	utils.NodeLog("worker", "Registered consumer for queue %s", n.Queue.Work.Name)
	// Queue Message Handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		select {
		case <-n.QueueReader:
			utils.NodeLog("worker", "Queue Reading goroutine canceled")
			return
		case d := <-msgs:
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
				utils.NodeLog("worker", "Computing Map Job (length %d)", len(job.MapData))
				result.Values = workerMap(n, job.MapData)
				utils.NodeLog("worker", "Completed Map Job")
			} else if job.Type == 1 {
				utils.NodeLog("worker", "Computing Reduce Job (length %d)", len(job.ReduceData))
				result.Values = workerReduce(n, job.ReduceData)
				utils.NodeLog("worker", "Completed Reduce Job")
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
}

func workerMap(n *Node, subGraph map[int32]*proto.Map) map[int32]float64 {
	contributions := make(map[int32]float64)
	for id, u := range subGraph {
		for _, v := range u.InLinks {
			contributions[id] += v.Rank / float64(v.Outlinks)
		}
	}
	return contributions
}

func workerReduce(n *Node, reduce map[int32]*proto.Reduce) map[int32]float64 {
	ranks := make(map[int32]float64)
	for id, v := range reduce {
		ranks[id] = n.State.C*v.Sum + (1-n.State.C)*v.E
	}
	return ranks
}

func workerHealthCheck(n *Node) {
	master, err := utils.NodeCall(n.Master)
	if err != nil {
		// Master didn't respond -> assuming crash
		utils.NodeLog("worker", "Failed to connect to master. Starting a new election")
		workerCandidacy(n)
		return
	}
	defer master.Close()
	health, err := master.Client.HealthCheck(
		master.Ctx,
		&wrapperspb.StringValue{Value: n.Connection},
	)
	if err != nil {
		// Master didn't respond -> assuming crash
		utils.NodeLog("worker", "Failed to get master response. Starting a new election")
		workerCandidacy(n)
		return
	}
	// No error detected -> master is still valid
	if state := health.GetState(); state != nil {
		// Last state update was missed -> loading now
		n.State = state
	}
}

func workerCandidacy(n *Node) {
	now := time.Now().UnixMilli()
	candidacy := &proto.Candidacy{Connection: n.Connection, Timestamp: now}
	n.Candidacy = now
	toRemove := make(map[int]bool)
	elected := true
	for i, v := range n.State.Others {
		// Skip connection to this node
		if v == n.Connection {
			// Remove this node
			toRemove[i] = true
			continue
		}
		utils.NodeLog("worker", "Contacting worker node: %s", v)
		worker, err := utils.NodeCall(v)
		if err != nil {
			utils.NodeLog("worker", "[WARN] Worker %s crashed", v)
			toRemove[i] = true
			continue
		}
		defer worker.Close()
		ack, err := worker.Client.MasterCandidate(worker.Ctx, candidacy)
		if err != nil {
			utils.NodeLog("worker", "[WARN] Worker %s crashed", v)
			toRemove[i] = true
			continue
		}
		// NACK -> there is an older candidacy
		if !ack.Value {
			utils.NodeLog("worker", "%s has an older candidacy. Withdrawing from election", v)
			elected = false
			break
		}
	}
	// Remove crashed workers from state
	var newWorkers []string
	for i, v := range n.State.Others {
		if !toRemove[i] {
			newWorkers = append(newWorkers, v)
		}
	}
	n.State.Others = newWorkers
	if elected {
		utils.NodeLog("worker", "Elected as new master")
		// Stop goroutines
		n.QueueReader <- true
		// Empty queues
		_, err := n.Queue.Channel.QueuePurge(n.Queue.Work.Name, true)
		utils.FailOnError("Failed to empty %s queue", err, n.Queue.Work.Name)
		_, err = n.Queue.Channel.QueuePurge(n.Queue.Result.Name, true)
		utils.FailOnError("Failed to empty %s queue", err, n.Queue.Result.Name)
		// Switch to master
		n.Role = Master
		// Start master update
		n.Update()
	}
}
