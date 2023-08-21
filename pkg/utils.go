package pkg

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/lioia/distributed-pagerank/pkg/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client[T interface{}] struct {
	Conn       *grpc.ClientConn
	Client     T
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

func NodeCall(url string) (Client[services.NodeClient], error) {
	var clientInfo Client[services.NodeClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := services.NewNodeClient(conn)
	if err != nil {
		return clientInfo, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	clientInfo.Conn = conn
	clientInfo.Client = client
	clientInfo.Ctx = ctx
	clientInfo.CancelFunc = cancel
	return clientInfo, nil
}

// https://stackoverflow.com/a/23558495
func GetAddressFromInterfaces() (string, error) {
	var address string
	interfaces, err := net.Interfaces()
	if err != nil {
		return address, err
	}

	for _, inter := range interfaces {
		addrs, err := inter.Addrs()
		if err != nil {
			continue // skip this interface
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				address = v.IP.String()
			case *net.IPAddr:
				address = v.IP.String()
			}
		}
	}
	return address, nil
}

func Url(conn *services.ConnectionInfo) string {
	return fmt.Sprintf("%s:%d", conn.Address, conn.Port)
}
