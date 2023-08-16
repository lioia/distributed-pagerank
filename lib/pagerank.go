package lib

import (
	"fmt"
	"math"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func (g *Graph) SingleNodePageRank(dampingFactor, threshold float64) {
	for i := 0; i < 100; i++ {
		sum := make(map[int32]float64)
		for _, u := range *g {
			// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
			nV := float64(len(u.OutLinks))
			for _, v := range u.OutLinks {
				sum[v.ID] += u.Rank / nV
			}
		}

		// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
		convergenceDiff := 0.0
		for id, node := range *g {
			oldRank := node.Rank
			newRank := dampingFactor*sum[id] + (1-dampingFactor)*node.EValue
			convergenceDiff += math.Abs(newRank - oldRank)
			(*g)[id].Rank = newRank
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}

func (g *Graph) GoroutinesPageRank(dampingFactor, threshold float64) {
	contributionChannel := make(chan map[int32]float64)
	convergenceChannel := make(chan float64)
	for i := 0; i < 100; i++ {
		sum := make(map[int32]float64)
		// Map
		for _, u := range *g {
			go goroutineMap(u, contributionChannel)
			contributions := <-contributionChannel
			for id := range contributions {
				sum[id] += contributions[id]
			}
		}

		// Reduce
		convergenceDiff := 0.0
		for id, node := range *g {
			go goroutineReduce(node, sum[id], dampingFactor, convergenceChannel)
			convergenceDiff += <-convergenceChannel
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}

func goroutineMap(u *GraphNode, channel chan map[int32]float64) {
	channel <- u.Map()
}

func goroutineReduce(node *GraphNode, sum, dampingFactor float64,
	diffChannel chan float64) {
	diffChannel <- node.Reduce(sum, dampingFactor)
}
