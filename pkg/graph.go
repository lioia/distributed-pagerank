package pkg

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/lioia/distributed-pagerank/proto"
)

func LoadGraphFromBytes(contents []byte) (*proto.Graph, error) {
	g := make(map[int32]*proto.GraphNode)
	// Split file contents in lines (based on newline delimiter)
	lines := strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n")
	for _, line := range lines {
		from, to, skip, err := convertLine(line)
		// There was an error loading the line
		if err != nil {
			return nil, err
		}
		// Comment line -> no new node to add
		if skip {
			continue
		}
		// First time encoutering this node, so it has to be created
		if g[from] == nil {
			g[from] = &proto.GraphNode{Id: from}
		}
		if g[to] == nil {
			g[to] = &proto.GraphNode{Id: to}
		}
		// Adding the outlink to the current node
		g[from].OutLinks = append(g[from].OutLinks, to)
	}

	// Initialize ranks and e values
	initialRank := 1.0 / float64(len(g))
	total := 0.0
	for id := range g {
		probability := rand.Float64()
		g[id].EValue = probability
		g[id].Rank = initialRank
		total += probability
	}

	// Normalize probability
	for id := range g {
		g[id].EValue /= total
	}
	return &proto.Graph{Graph: g}, nil

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
		return 0, 0, false, fmt.Errorf("Could not convert FromNode %s", tokens[0])
	}
	to, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, 0, false, fmt.Errorf("Could not convert ToNode %s", tokens[1])
	}
	return int32(from), int32(to), false, nil
}
