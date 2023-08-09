package lib

import (
	"fmt"
	"math"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func PageRank(graph map[int]*Node, dampingFactor, threshold float64) {
	sum := make(map[int]float64)
	for _, node := range graph {
		// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
		nV := float64(len(node.OutLinks))
		for _, v := range node.OutLinks {
			currentVRank := graph[v].Rank
			sum[v] += currentVRank / nV
		}
	}

	// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
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
