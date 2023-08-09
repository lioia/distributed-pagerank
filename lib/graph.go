package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Node struct {
	ID       int     // Node identifier
	OutLinks []int   // Connected nodes
	Rank     float64 // Current PageRank
	EValue   float64 // E probability vector
}

func PrintMatrix(graph [][]float64) {
	size := len(graph)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			fmt.Printf("%.4f ", graph[i][j])
		}
		fmt.Println()
	}
}

func PrintGraphAdjacencyList(graph map[int]*Node) {
	for _, node := range graph {
		fmt.Printf("%d -> ", node.ID)
		for _, v := range node.OutLinks {
			fmt.Printf("%d ", v)
		}
		fmt.Println()
	}
}

func PrintRanksAdjacencyList(graph map[int]*Node) {
	for _, node := range graph {
		fmt.Printf("Node %d with rank: %.4f\n", node.ID, node.Rank)
	}
}

// Load file and creates adjacency matrix
// Adjacency Matrix A of G=(V, E) defined as follow: a_(i j) = 1 iff (i j) in E
// TODO: it currently requires a double pass of the file, maybe it can be created in a single pass
func LoadAdjacencyMatrixFromFile(path string) ([][]float64, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s", path)
	}

	numberOfNodes := 0
	lines := strings.Split(strings.ReplaceAll(string(bytes), "\r\n", "\n"), "\n")
	for _, line := range lines {
		fromNode, toNode, skip, err := convertLine(line)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		if numberOfNodes < fromNode {
			numberOfNodes = fromNode
		}
		if numberOfNodes < toNode {
			numberOfNodes = toNode
		}
	}

	matrix := make([][]float64, numberOfNodes)
	for i := 0; i < numberOfNodes; i++ {
		matrix[i] = make([]float64, numberOfNodes)
	}

	for _, line := range lines {
		fromNode, toNode, skip, err := convertLine(line)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		matrix[fromNode-1][toNode-1] = 1.0
	}

	return matrix, nil
}

func LoadAdjacencyListFromFile(path string) (map[int]*Node, error) {
	graph := make(map[int]*Node)
	// Read file
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s", path)
	}
	// Split file contents in lines (based on newline delimiter)
	lines := strings.Split(strings.ReplaceAll(string(bytes), "\r\n", "\n"), "\n")
	for _, line := range lines {
		fromNode, toNode, skip, err := convertLine(line)
		// There was an error loading the line
		if err != nil {
			return nil, err
		}
		// Comment line -> no new node to add
		if skip {
			continue
		}
		// First time encoutering this node, so it has to be created
		if graph[fromNode] == nil {
			graph[fromNode] = &Node{ID: fromNode, Rank: 1.0}
		}
		// Adding the outlink to the current node
		graph[fromNode].OutLinks = append(graph[fromNode].OutLinks, toNode)
	}
	return graph, nil
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
