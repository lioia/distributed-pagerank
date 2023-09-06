package utils

import (
	"fmt"
	"log"
)

var nodeLog bool
var serverLog bool

func InitLog(node, server bool) {
	nodeLog = node
	serverLog = server
}

func ServerLog(format string, v ...any) {
	if serverLog {
		log.Printf("INFO Server: %s", fmt.Sprintf(format, v...))
	}
}

func NodeLog(role string, format string, v ...any) {
	if nodeLog {
		log.Printf("INFO Compute %s: %s", role, fmt.Sprintf(format, v...))
	}
}

func WarnLog(role string, format string, v ...any) {
	log.Printf("WARN %s: %s", role, fmt.Sprintf(format, v...))
}
