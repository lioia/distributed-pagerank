package lib

import "math/rand"

func InitializeEValues(graph map[int]*Node) {
	var total float64

	for id := range graph {
		probability := rand.Float64()
		graph[id].EValue = probability
		total += probability
	}

	// Normalize
	for id := range graph {
		graph[id].EValue /= total
	}
}
