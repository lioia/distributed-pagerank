package lib

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientInformation[T interface{}] struct {
	Conn       *grpc.ClientConn
	Client     T
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

// NOTE: MasterClientCall, Layer1Clientcall, Layer2ClientCall could be removed
// once this proposal https://github.com/golang/go/issues/45380
// or similar proposals will be implemented
// (missing type switch in type parameters)

// Utility function to create a grpc client to `url`
// User has to defer conn.close() and defer cancel()
func MasterClientCall(url string) (ClientInformation[MasterNodeClient], error) {
	var clientInfo ClientInformation[MasterNodeClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := NewMasterNodeClient(conn)
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

// Utility function to create a grpc client to `url`
// User has to defer conn.close() and defer cancel()
func Layer1ClientCall(url string) (ClientInformation[Layer1NodeClient], error) {
	var clientInfo ClientInformation[Layer1NodeClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := NewLayer1NodeClient(conn)
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

// Utility function to create a grpc client to `url`
// User has to defer conn.close() and defer cancel()
func Layer2ClientCall(url string) (ClientInformation[Layer2NodeClient], error) {
	var clientInfo ClientInformation[Layer2NodeClient]
	conn, err := grpc.Dial(
		url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return clientInfo, err
	}
	client := NewLayer2NodeClient(conn)
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
