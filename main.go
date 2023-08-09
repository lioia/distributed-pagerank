package main

import (
	"github.com/lioia/distributed-pagerank/lib"
)

func main() {
	dampingFactor := 0.85
	threshold := 0.0001
	list, err := lib.LoadAdjacencyListFromFile("graph.txt")
	if err != nil {
		panic(err)
	}
	lib.PrintGraphAdjacencyList(list)
	lib.InitializeEValues(list)
	lib.PageRank(list, dampingFactor, threshold)
	lib.PrintRanksAdjacencyList(list)
}
