package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lioia/distributed-pagerank/pkg"
	"github.com/lioia/distributed-pagerank/pkg/nodes"
	"github.com/lioia/distributed-pagerank/pkg/services"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NodeServerImpl struct {
	Node *nodes.Node
	services.UnimplementedNodeServer
}

func (s *NodeServerImpl) StateUpdate(context.Context, *emptypb.Empty) (*services.State, error) {
	state := services.State{
		Phase:  int32(s.Node.Phase),
		Role:   int32(s.Node.Role),
		Graph:  s.Node.Graph,
		Others: s.Node.Others,
	}
	return &state, nil
}

func (s *NodeServerImpl) NodeJoin(context.Context, *emptypb.Empty) (*services.Constants, error) {
	if s.Node.Role == nodes.Worker {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.UpperLayer)
	}
	constants := services.Constants{
		WorkQueue:   s.Node.Queue.Work.Name,
		ResultQueue: s.Node.Queue.Result.Name,
		C:           s.Node.C,
		Threshold:   s.Node.Threshold,
	}
	return &constants, nil
}

func (s *NodeServerImpl) UploadGraph(_ context.Context, in *services.GraphUpload) (*services.Graph, error) {
	if s.Node.Role != nodes.Master {
		return nil, fmt.Errorf("This node cannot fulfill this request. Contact master node at: %s", s.Node.UpperLayer)
	}
	graph, err := pkg.LoadGraphFromBytes(in.Contents)
	if err != nil {
		return nil, fmt.Errorf("Could not load graph: %v", err)
	}
	// No other node in the network -> calculating PageRank on this node
	if len(s.Node.Others) == 0 {
		pkg.SingleNodePageRank(graph, s.Node.C, s.Node.Threshold)
		return graph, nil
	}
	// Divide Graph in SubGraphs
	numberOfSubGraphs := len(s.Node.Graph.Graph) / len(s.Node.Others)
	subGraphs := make([]*services.Graph, numberOfSubGraphs)
	index := 0
	for id, node := range s.Node.Graph.Graph {
		subGraphs[index/numberOfSubGraphs].Graph[id] = node
		index += 1
	}
	// Send subgraph to work queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, subGraph := range subGraphs {
		data, err := proto.Marshal(subGraph)
		if err != nil {
			// TODO: empty queue
			return nil, err
		}
		err = s.Node.Queue.Channel.PublishWithContext(ctx,
			"",
			s.Node.Queue.Work.Name, // routing key
			false,                  // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/x-protobuf",
				Body:         data,
			})
		if err != nil {
			// TODO: empty queue
			return nil, err
		}
	}
	// Switch to Map phase
	s.Node.Phase = nodes.Map
	// Graph was successfully uploaded and computation has started
	return nil, nil
}
