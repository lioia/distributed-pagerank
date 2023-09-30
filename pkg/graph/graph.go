package graph

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/lioia/distributed-pagerank/proto"
)

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

func Generate(numberOfNodes, maxNumberOfEdges int32) map[int32]*proto.GraphNode {
	graph := make(map[int32]*proto.GraphNode)
	numberOfOutlinks := make(map[int32]int32)
	for from := 0; from < int(numberOfNodes); from++ {
		// Generate number of edges for node from
		outlinks := rand.Int31n(maxNumberOfEdges) + 1
		for j := 0; j < int(outlinks); j++ {
			// Generate node to
			to := rand.Int31n(numberOfNodes)
			for to == int32(from) {
				to = rand.Int31n(numberOfNodes)
			}
			// Initialize from and to if they don't exist
			if graph[int32(from)] == nil {
				graph[int32(from)] = &proto.GraphNode{
					InLinks: make(map[int32]*proto.GraphNodeInfo),
				}
			}
			if graph[int32(to)] == nil {
				graph[int32(to)] = &proto.GraphNode{
					InLinks: make(map[int32]*proto.GraphNodeInfo),
				}
			}
			// Edge: from -> to
			graph[to].InLinks[int32(from)] = &proto.GraphNodeInfo{}
		}
	}

	// Ensure connectivity by adding an edge between consecutive nodes
	for i := range graph {
		if i == 0 {
			// Skip first node
			continue
		}
		if graph[i].InLinks[i-1] == nil {
			graph[i].InLinks[i-1] = &proto.GraphNodeInfo{}
		}
	}

	// Find out number of outlinks
	for _, v := range graph {
		for j := range v.InLinks {
			numberOfOutlinks[j] += 1
		}
	}

	// Set default values
	initialRank := 1.0 / float64(numberOfNodes)
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
	return graph
}

func ConvertToSvg(g map[int32]*proto.GraphNode) (string, error) {
	gviz := graphviz.New()
	graph, err := gviz.Graph()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := graph.Close(); err != nil {
			log.Fatal(err)
		}
		gviz.Close()
	}()
	nodes := make(map[int32]*cgraph.Node)
	// Create Nodes
	for i := range g {
		n, err := graph.CreateNode(fmt.Sprintf("%d", i))
		if err != nil {
			return "", err
		}
		nodes[i] = n
	}
	// Create Edges
	for i, v := range g {
		n := nodes[i]
		for j := range v.InLinks {
			m := nodes[j]
			_, err := graph.CreateEdge(fmt.Sprintf("%d -> %d", j, i), m, n)
			if err != nil {
				return "", err
			}
		}
	}
	var buf bytes.Buffer
	if err := gviz.Render(graph, graphviz.SVG, &buf); err != nil {
		return "", err
	}
	svg := fmt.Sprintf("<svg%s", strings.Split(buf.String(), "<svg")[1])
	return svg, nil
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
