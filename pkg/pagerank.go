package pkg

import (
	"fmt"
	"math"

	"github.com/lioia/distributed-pagerank/proto"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func SingleNodePageRank(graph map[int32]*proto.GraphNode, c, threshold float64) {
	for i := 0; i < 100; i++ {
		sum := make(map[int32]float64)
		for _, u := range graph {
			// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
			nV := float64(len(u.OutLinks))
			for _, v := range u.OutLinks {
				sum[v] += u.Rank / nV
			}
		}

		// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
		convergenceDiff := 0.0
		for id, node := range graph {
			oldRank := node.Rank
			newRank := c*sum[id] + (1-c)*node.EValue
			convergenceDiff += math.Abs(newRank - oldRank)
			graph[id].Rank = newRank
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}
