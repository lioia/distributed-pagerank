package main

import (
	"github.com/lioia/distributed-pagerank/lib"
)

func main() {
	// TODO: load from configuration file
	dampingFactor := 0.85
	threshold := 0.0001

	var graph lib.Graph
	if err := graph.LoadFromFile("graph.txt"); err != nil {
		panic(err)
	}
	graph.PageRank(dampingFactor, threshold)
	graph.Print()
}
