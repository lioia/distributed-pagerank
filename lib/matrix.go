package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Print(matrix [][]int) {
	for i := range matrix {
		for j := range matrix[i] {
			fmt.Printf("%d ", matrix[i][j])
		}
		fmt.Println()
	}
}

// Load file and creates adjacency matrix
// Adjacency Matrix A of G=(V, E) defined as follow: a_(i j) = 1 iff (i j) in E
// TODO: it currently requires a double pass of the file, maybe it can be created in a single pass
func LoadFromFile(path string) ([][]int, error) {
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

	matrix := make([][]int, numberOfNodes)
	for i := 0; i < numberOfNodes; i++ {
		matrix[i] = make([]int, numberOfNodes)
	}

	for _, line := range lines {
		fromNode, toNode, skip, err := convertLine(line)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		matrix[fromNode-1][toNode-1] = 1
	}

	return matrix, nil
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
