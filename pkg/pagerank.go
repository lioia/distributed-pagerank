package pkg

import (
	"fmt"
	"math"

	"github.com/lioia/distributed-pagerank/pkg/services"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func SingleNodePageRank(graph *services.Graph, c, threshold float64) {
	for i := 0; i < 100; i++ {
		sum := make(map[int32]float64)
		for _, u := range graph.Graph {
			// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
			nV := float64(len(u.OutLinks))
			for _, v := range u.OutLinks {
				sum[v.Id] += u.Rank / nV
			}
		}

		// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
		convergenceDiff := 0.0
		for id, node := range graph.Graph {
			oldRank := node.Rank
			newRank := c*sum[id] + (1-c)*node.EValue
			convergenceDiff += math.Abs(newRank - oldRank)
			graph.Graph[id].Rank = newRank
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}

func ComputeMap(u *services.GraphNode) map[int32]float64 {
	contributions := make(map[int32]float64)
	nV := float64(len(u.OutLinks))
	for _, v := range u.OutLinks {
		contributions[v.Id] = u.Rank / nV
	}
	return contributions
}

func ComputeReduce(u *services.GraphNode, sum, c float64) float64 {
	return c*sum + (1-c)*u.EValue
}