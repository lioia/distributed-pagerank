package graph

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lioia/distributed-pagerank/pkg/proto"
)

func Write(output string, graph map[int32]*proto.GraphNode) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	var contents string
	for id, v := range graph {
		contents = fmt.Sprintf("%sNode %d with rank %f\n", contents, id, v.Rank)
	}
	_, err = file.WriteString(contents)
	return err
}

func LoadGraphResource(resource string) (g map[int32]*proto.GraphNode, err error) {
	var bytes []byte
	// Check if it's a network resource or a local one
	if strings.HasPrefix(resource, "http") {
		// Loading file from network
		var resp *http.Response
		resp, err = http.Get(resource)
		if err != nil {
			log.Printf("Could not load network file at %s: %v", resource, err)
			return nil, err
		}
		defer resp.Body.Close()
		// Read response body
		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Could not load body from request: %v", err)
			return nil, err
		}
	} else {
		// Loading file from local filesystem
		bytes, err = os.ReadFile(resource)
		if err != nil {
			log.Printf("Could not read graph at %s: %v", resource, err)
			return nil, err
		}
	}
	// Parse graph file into graph representation
	g, err = LoadGraphFromBytes(bytes)
	if err != nil {
		log.Printf("Could not load graph from %s: %v", resource, err)
		return nil, err
	}
	return g, nil
}

func LoadGraphFromBytes(contents []byte) (map[int32]*proto.GraphNode, error) {
	// Split file contents in lines (based on newline delimiter)
	graph := make(map[int32]*proto.GraphNode)
	numberOfOutlinks := make(map[int32]int32)
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
		if graph[from] == nil {
			graph[from] = &proto.GraphNode{
				InLinks: make(map[int32]*proto.GraphNodeInfo),
			}
			numberOfOutlinks[from] = 0
		}
		if graph[to] == nil {
			graph[to] = &proto.GraphNode{
				InLinks: make(map[int32]*proto.GraphNodeInfo),
			}
		}
		graph[to].InLinks[from] = &proto.GraphNodeInfo{}
		numberOfOutlinks[from] += 1
	}
	initialRank := 1.0 / float64(len(graph))
	total := 0.0
	for _, u := range graph {
		probability := rand.Float64()
		u.Rank = initialRank
		u.E = probability
		total += probability
		for j, v := range u.InLinks {
			v.Rank = initialRank
			v.Outlinks = numberOfOutlinks[j]
		}
	}

	// Normalize E values (sum i equal to 1)
	for _, v := range graph {
		v.E /= total
	}

	return graph, nil
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
