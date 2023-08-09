package lib

import (
	"fmt"
	"math"
)

// R_(i + 1) (u) = c sum_(v in B_u) (R_i(v) / N_v) + (1 - c)E(u)
func (g *Graph) PageRank(dampingFactor, threshold float64) {
	for i := 0; i < 100; i++ {
		sum := make(map[int]float64)
		for _, u := range g.Nodes {
			// Map Phase: sum_(v in B_u) (R_i(v) / N_v)
			nV := float64(len(u.OutLinks))
			for _, v := range u.OutLinks {
				sum[v.ID] += u.Rank / nV
			}
		}

		// Reduce phase (convergence check): R_(i + 1) (u) = c * sum + (1-c)*E(u)
		convergenceDiff := 0.0
		for id, node := range g.Nodes {
			oldRank := node.Rank
			newRank := dampingFactor*sum[id] + (1-dampingFactor)*node.EValue
			convergenceDiff += math.Abs(newRank - oldRank)
			g.Nodes[id].Rank = newRank
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}

func (g *Graph) GoroutinesPageRank(dampingFactor, threshold float64) {
	contributionChannel := make(chan map[int]float64)
	convergenceChannel := make(chan float64)
	for i := 0; i < 100; i++ {
		sum := make(map[int]float64)
		// Map
		for _, u := range g.Nodes {
			go mapper(u, contributionChannel)
			contributions := <-contributionChannel
			for id := range contributions {
				sum[id] += contributions[id]
			}
		}

		// Reduce
		convergenceDiff := 0.0
		for id, node := range g.Nodes {
			go reducer(node, sum[id], dampingFactor, convergenceChannel)
			convergenceDiff += <-convergenceChannel
		}
		if convergenceDiff < threshold {
			fmt.Printf("Converged after %d iteration(s)\n", i+1)
			break
		}
	}
}

func mapper(u *Node, channel chan map[int]float64) {
	contributions := make(map[int]float64)
	nV := float64(len(u.OutLinks))
	for _, v := range u.OutLinks {
		contributions[v.ID] = u.Rank / nV
	}
	channel <- contributions
}

func reducer(node *Node, sum, dampingFactor float64,
	convergenceDiff chan float64) {
	oldRank := node.Rank
	newRank := dampingFactor*sum + (1-dampingFactor)*node.EValue
	diff := math.Abs(newRank - oldRank)
	node.Rank = newRank
	convergenceDiff <- diff
}
