package lib

import (
	"fmt"
	"math"
)

func PageRank(graph map[int]*Node, dampingFactor, threshold float64) {
	// Map Phase
	sum := make(map[int]float64)
	for _, node := range graph {
		nV := len(node.OutLinks)
		for _, outLink := range node.OutLinks {
			sum[outLink] += node.Rank / float64(nV)
		}
	}

	// Reduce phase (convergence check)
	converged := true
	for id, node := range graph {
		oldRank := node.Rank
		newRank := dampingFactor*sum[id] + (1-dampingFactor)*node.EValue
		if math.Abs(newRank-oldRank) > threshold {
			converged = false
		}
		graph[id].Rank = newRank
	}
	if converged {
		fmt.Println("Converged: no need to continue")
	}

	fmt.Printf("Converged: %t\n", converged)
}
