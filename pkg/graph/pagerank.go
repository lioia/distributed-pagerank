package graph

import (
	"math"

	"github.com/lioia/distributed-pagerank/pkg/utils"
	"github.com/lioia/distributed-pagerank/proto"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func SingleNodePageRank(graph map[int32]*proto.GraphNode, c, threshold float64) int32 {
	for i := 0; i < 100; i++ {
		sum := make(map[int32]float64, len(graph))
		for id, u := range graph {
			// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
			for _, v := range u.InLinks {
				sum[id] += v.Rank / float64(v.Outlinks)
			}
		}

		// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
		convergenceDiff := 0.0
		for id, node := range graph {
			oldRank := node.Rank
			newRank := c*sum[id] + (1-c)*node.E
			convergenceDiff += math.Abs(newRank - oldRank)
			graph[id].Rank = newRank
		}
		// Update InLinks rank
		for _, node := range graph {
			for j, in := range node.InLinks {
				in.Rank = graph[j].Rank
			}
		}

		if convergenceDiff < threshold {
			utils.NodeLog("master", "Convergence check success (%d iterations)", i+1)
			// Normalize values
			rankSum := 0.0
			for _, node := range graph {
				rankSum += node.Rank
			}
			for i := range graph {
				graph[i].Rank /= rankSum
			}
			return int32(i)
		} else {
			utils.NodeLog("master", "Convergence check failed (%f)", convergenceDiff)
		}
	}
	return 100
}
