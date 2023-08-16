package lib

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

type Graph map[int32]*GraphNode

func (u *GraphNode) Map() map[int32]float64 {
	contributions := make(map[int32]float64)
	nV := float64(len(u.OutLinks))
	for _, v := range u.OutLinks {
		contributions[v.ID] = u.Rank / nV
	}
	return contributions
}

func (u *GraphNode) Reduce(sum, dampingFactor float64) float64 {
	oldRank := u.Rank
	newRank := dampingFactor*sum + (1-dampingFactor)*u.EValue
	diff := math.Abs(newRank - oldRank)
	u.Rank = newRank
	return diff
}

func (g *Graph) Print() {
	for _, node := range *g {
		fmt.Printf("Node %d with rank %.4f and OutLinks ", node.ID, node.Rank)
		for _, v := range node.OutLinks {
			fmt.Printf("%d ", v.ID)
		}
		fmt.Println()
	}
}

func (g *Graph) LoadFromFile(path string) error {
	// Read bytes from file
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open file %s", path)
	}
	return g.LoadFromBytes(bytes)
}

func (g *Graph) LoadFromBytes(bytes []byte) error {
	// Split file contents in lines (based on newline delimiter)
	lines := strings.Split(strings.ReplaceAll(string(bytes), "\r\n", "\n"), "\n")
	for _, line := range lines {
		from, to, skip, err := convertLine(line)
		// There was an error loading the line
		if err != nil {
			return err
		}
		// Comment line -> no new node to add
		if skip {
			continue
		}
		// First time encoutering this node, so it has to be created
		if (*g)[from] == nil {
			(*g)[from] = &GraphNode{
				ID:       from,
				OutLinks: make(Graph),
			}
		}
		if (*g)[to] == nil {
			(*g)[to] = &GraphNode{
				ID:       to,
				OutLinks: make(Graph),
			}
		}
		// Adding the outlink to the current node
		(*g)[from].OutLinks[to] = (*g)[to]
	}

	// Initialize ranks and e values
	initialRank := 1.0 / float64(len(*g))
	total := 0.0
	for id := range *g {
		probability := rand.Float64()
		(*g)[id].EValue = probability
		(*g)[id].Rank = initialRank
		total += probability
	}

	// Normalize probability
	for id := range *g {
		(*g)[id].EValue /= total
	}
	return nil
}

func convertLine(line string) (int32, int32, bool, error) {
	// Skip comment lines
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || line == "" {
		return 0, 0, true, nil
	}
	// Convert line to csv format
	line = strings.Replace(line, " ", ",", 1)
	// Split line in FromNode and ToNode
	tokens := strings.Split(line, ",")
	from, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, 0, false, fmt.Errorf("could not convert FromNode %s", tokens[0])
	}
	to, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, 0, false, fmt.Errorf("could not convert ToNode %s", tokens[1])
	}
	return int32(from), int32(to), false, nil
}
