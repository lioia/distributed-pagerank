package main

import (
	"fmt"

	"github.com/lioia/distributed-pagerank/lib"
)

type Node interface {
	Init(info *lib.Info) error
}

type BaseNode struct {
	Layer   int32
	Address string
	Port    int32
}

type Layer1Node struct {
	BaseNode                            // Base node information
	FirstNode     *lib.ConnectionInfo   // Connection info of first node
	Layer1s       []*lib.ConnectionInfo // Connection info for other first layer nodes
	NumberOfNodes []int32               // # nodes for every first layer
	Layer2s       []*lib.ConnectionInfo // Connection info for second layer nodes
	// TODO: data
}

type Layer2Node struct {
	BaseNode                     // Base node information
	Layer1   *lib.ConnectionInfo // Assigned first layer connection info
	Phase    int32               // Current computation phase: 0 = wait; 1 = map; 2 = reduce
	// TODO: data
}

func (n *Layer1Node) Init(info *lib.Info) error {
	for _, v := range info.GetLayer1S() {
		// Save information on the other layer 1 nodes
		n.Layer1s = append(n.Layer1s, v)
		// Contact other layer 1 nodes
		layer1Url := fmt.Sprintf("%s:%d", v.Address, v.Port)
		clientInfo, err := lib.ClientCall(layer1Url)
		// FIXME: error handling
		if err != nil {
			return err
		}
		announceMsg := lib.AnnounceMessage{
			LayerNumber: 1,
			Connection: &lib.ConnectionInfo{
				Address: n.Address,
				Port:    n.Port,
			},
		}
		_, err = clientInfo.Client.Announce(clientInfo.Context, &announceMsg)
		// FIXME: error handling
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Layer2Node) Init(info *lib.Info) error {
	n.Layer1 = info.GetAssigned()
	n.Phase = 0 // waiting for command phase
	layer1Url := fmt.Sprintf("%s:%d", n.Layer1.Address, n.Layer1.Port)
	clientInfo, err := lib.ClientCall(layer1Url)
	// FIXME: error handling
	if err != nil {
		return err
	}
	announceMsg := lib.AnnounceMessage{
		LayerNumber: 2,
		Connection: &lib.ConnectionInfo{
			Address: n.Address,
			Port:    n.Port,
		},
	}
	_, err = clientInfo.Client.Announce(clientInfo.Context, &announceMsg)
	// FIXME: error handling
	if err != nil {
		return err
	}
	return nil
}
