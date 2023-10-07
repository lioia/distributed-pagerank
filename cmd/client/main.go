package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lioia/distributed-pagerank/pkg/node"
	"github.com/lioia/distributed-pagerank/pkg/utils"
	"github.com/lioia/distributed-pagerank/proto"
	"google.golang.org/grpc"
)

type TemplateRenderer struct {
	tmpl *template.Template
}

func (t TemplateRenderer) Render(w io.Writer, name string, data interface{}, e echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type IndexPage struct {
	Status     string
	Master     string
	Svg        template.HTML
	Values     map[int32]float64
	Error      string
	FormErrors map[string]string
}

func main() {
	_ = godotenv.Load()
	host, err := utils.ReadStringEnvVar("HOST")
	utils.FailOnError("Failed to load environment variables", err)
	rpcPort, err := utils.ReadIntEnvVar("RPC_PORT")
	utils.FailOnError("Failed to load environment variables", err)
	webPort := utils.ReadIntEnvVarOr("WEB_PORT", 80)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, rpcPort))
	utils.FailOnError("Failed to listen for node server", err)
	fmt.Printf("Starting client API server on: %s:%d\n", host, rpcPort)

	ranks := make(chan *proto.Ranks)
	iteration := make(chan int32)
	// Create gRPC server
	go func() {
		defer lis.Close()
		server := grpc.NewServer()
		proto.RegisterAPIServer(server, &node.ApiServerImpl{
			Ranks:      ranks,
			Iterations: iteration,
		})
		err = server.Serve(lis)
		utils.FailOnError("Failed to serve", err)
	}()

	tmpls, err := template.ParseFiles(
		"public/index.html",
		"public/tmpl.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	e := echo.New()
	e.Renderer = TemplateRenderer{tmpl: tmpls}
	e.Use(middleware.Logger())

	e.GET("/", index)

	e.POST("/ranks/new", func(c echo.Context) error {
		return newRanks(c, fmt.Sprintf("%s:%d", host, rpcPort))
	})
	e.GET("/ranks", func(c echo.Context) error {
		return sseRanks(c, ranks, iteration, tmpls)
	})
	log.Println("Starting web server")
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", webPort)))
}

func index(c echo.Context) error {
	return c.Render(200, "index.html", nil)
}

func sseRanks(c echo.Context, ranks chan *proto.Ranks, iteration chan int32, tmpls *template.Template) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case value := <-iteration:
			var msgBuffer bytes.Buffer
			var msg string
			err := tmpls.ExecuteTemplate(&msgBuffer, "status", IndexPage{
				Status: fmt.Sprintf("%d", value),
			})
			if err != nil {
				msg = "Failed to read values"
			} else {
				msg = strings.ReplaceAll(msgBuffer.String(), "\n", "")
			}
			fmt.Fprintf(c.Response().Writer, "data: %s\n\n", msg)
			return nil
		case values := <-ranks:
			var msgBuffer bytes.Buffer
			var msg string
			err := tmpls.ExecuteTemplate(&msgBuffer, "ranks", IndexPage{
				Values: values.Ranks,
				Master: values.Master,
				Status: values.Status,
				Svg:    template.HTML(values.Svg),
			})
			if err != nil {
				msg = "Failed to read values"
			} else {
				msg = strings.ReplaceAll(msgBuffer.String(), "\n", "")
			}
			fmt.Fprintf(c.Response().Writer, "data: %s\n\n", msg)
			return nil
		}
	}
}

func newRanks(ctx echo.Context, connection string) error {
	apiUrl := ctx.FormValue("api")
	cStr := ctx.FormValue("c")
	thresholdStr := ctx.FormValue("threshold")
	graph := ctx.FormValue("graph")
	numNodesStr := ctx.FormValue("numNodes")
	numNodes := 30
	numEdgesStr := ctx.FormValue("numEdges")
	numEdges := 5

	errors := make(map[string]string)

	if len(strings.Split(apiUrl, ":")) != 2 {
		errors["api"] = "Invalid API Url (Expecting host:port)"
	}
	c, err := strconv.ParseFloat(cStr, 64)
	if err != nil {
		errors["c"] = "Failed to parse as a number"
	}
	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		errors["threshold"] = "Failed to parse as a number"
	}
	if graph != "" && !strings.HasPrefix(graph, "http") {
		errors["graph"] = "Invalid Graph Resource"
	}
	if numNodesStr != "" {
		num, err := strconv.Atoi(numNodesStr)
		if err != nil {
			errors["numNodes"] = "Failed to parse as a number"
		} else if num >= 5 {
			numNodes = num
		}
	}
	numEdgesTemp, err := strconv.Atoi(numEdgesStr)
	if err == nil && numEdgesTemp >= 3 {
		numEdges = numEdgesTemp
	}

	if len(errors) > 0 {
		return ctx.Render(200, "ranks.new", IndexPage{FormErrors: errors})
	}

	configuration := proto.Configuration{
		C:          c,
		Threshold:  threshold,
		Connection: connection,
	}

	if graph != "" {
		configuration.Value = &proto.Configuration_Graph{Graph: graph}
	} else {
		configuration.Value = &proto.Configuration_RandomGraph{
			RandomGraph: &proto.RandomGraph{
				NumberOfNodes:    int32(numNodes),
				MaxNumberOfEdges: int32(numEdges),
			},
		}
	}

	api, err := utils.ApiCall(apiUrl)
	if err != nil {
		return ctx.Render(200, "ranks.new", IndexPage{
			Error: fmt.Sprintf("Failed to contact API: %v", err),
		})
	}
	_, err = api.Client.GraphUpload(api.Ctx, &configuration)
	if err != nil {
		return ctx.Render(200, "ranks.new", IndexPage{
			Error: fmt.Sprintf("Failed to call API: %v", err),
		})
	}

	return ctx.Render(200, "status", IndexPage{
		Status: "Calculating...",
	})
}
