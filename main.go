package main

import "github.com/lioia/distributed-pagerank/lib"

func main() {
	matrix, err := lib.LoadAdjacencyMatrixFromFile("graph.txt")
	if err != nil {
		panic(err)
	}
	lib.PrintMatrix(matrix)
	list, err := lib.LoadAdjacencyListFromFile("graph.txt")
	if err != nil {
		panic(err)
	}
	lib.PrintList(list)
}
