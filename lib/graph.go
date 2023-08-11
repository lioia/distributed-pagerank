package lib

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// Struct to _attach_ methods to
type Graph struct {
	Nodes map[int]*GraphNode
}

type GraphNode struct {
	ID       int                // Node identifier
	OutLinks map[int]*GraphNode // Nodes this node points to
	Rank     float64            // Current PageRank
	EValue   float64            // E probability vector
}

func (g *Graph) Print() {
	for i := 0; i < len(g.Nodes); i++ {
		node := g.Nodes[i]
		fmt.Printf("Node %d with rank %.4f and OutLinks ", node.ID, node.Rank)
		for _, v := range node.OutLinks {
			fmt.Printf("%d ", v.ID)
		}
		fmt.Println()
	}
}

func (g *Graph) LoadFromFile(path string) error {
	g.Nodes = make(map[int]*GraphNode)
	// Read file
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not open file %s", path)
	}
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
		if g.Nodes[from] == nil {
			g.Nodes[from] = &GraphNode{
				ID:       from,
				OutLinks: make(map[int]*GraphNode),
			}
		}
		if g.Nodes[to] == nil {
			g.Nodes[to] = &GraphNode{
				ID:       to,
				OutLinks: make(map[int]*GraphNode),
			}
		}
		// Adding the outlink to the current node
		g.Nodes[from].OutLinks[to] = g.Nodes[to]
	}

	// Initialize ranks and e values
	initialRank := 1.0 / float64(len(g.Nodes))
	total := 0.0
	for id := range g.Nodes {
		probability := rand.Float64()
		g.Nodes[id].EValue = probability
		g.Nodes[id].Rank = initialRank
		total += probability
	}

	// Normalize probability
	for id := range g.Nodes {
		g.Nodes[id].EValue /= total
	}
	return nil
}

func convertLine(line string) (int, int, bool, error) {
	// Skip comment lines
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || line == "" {
		return 0, 0, true, nil
	}
	// Convert line to csv format
	line = strings.Replace(line, " ", ",", 1)
	// Split line in FromNode and ToNode
	tokens := strings.Split(line, ",")
	fromNode, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, 0, false, fmt.Errorf("could not convert FromNode %s", tokens[0])
	}
	toNode, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, 0, false, fmt.Errorf("could not convert ToNode %s", tokens[1])
	}
	return fromNode, toNode, false, nil
}
