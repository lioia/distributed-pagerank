package lib

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientInformation struct {
	ClientConnection *grpc.ClientConn
	Client           NodeClient
	Context          context.Context
	CancelFunction   context.CancelFunc
}

// Utility function to create a grpc client to `url`
// User has to defer conn.close() and defer cancel()
func ClientCall(url string) (ClientInformation, error) {
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return ClientInformation{}, err
	}
	client := NewNodeClient(conn)
	if err != nil {
		return ClientInformation{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	clientInformation := ClientInformation{
		ClientConnection: conn,
		Client:           client,
		Context:          ctx,
		CancelFunction:   cancel,
	}
	return clientInformation, nil
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
