package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type EnvVars struct {
	Master      string
	Host        string
	Port        int
	RabbitHost  string
	RabbitUser  string
	RabbitPass  string
	WorkQueue   string
	ResultQueue string
	NodeLog     bool
	ServerLog   bool
}

func ReadEnvVars() EnvVars {
	// Loading .env file if it exists
	// It will not override already existing env vars
	_ = godotenv.Load()
	master, err := ReadStringEnvVar("MASTER")
	FailOnError("Failed to read environment variables", err)
	host := ReadStringEnvVarOr("HOST", "")
	port, err := ReadIntEnvVar("PORT")
	FailOnError("Failed to read environment variables", err)
	rabbitHost, err := ReadStringEnvVar("RABBIT_HOST")
	FailOnError("Failed to read environment variables", err)
	rabbitUser := ReadStringEnvVarOr("RABBIT_USER", "guest")
	rabbitPass := ReadStringEnvVarOr("RABBIT_PASSWORD", "guest")
	workQueue := ReadStringEnvVarOr("WORK_QUEUE", "work")
	resultQueue := ReadStringEnvVarOr("RESULT_QUEUE", "result")
	nodeLog := readBoolEnvVarOr("NODE_LOG", false)
	serverLog := readBoolEnvVarOr("SERVER_LOG", false)
	return EnvVars{
		Master: master, Host: host, Port: port,
		RabbitHost: rabbitHost, RabbitUser: rabbitUser, RabbitPass: rabbitPass,
		WorkQueue: workQueue, ResultQueue: resultQueue,
		NodeLog: nodeLog, ServerLog: serverLog,
	}
}

func ReadStringEnvVar(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s not set", name)
	}
	return value, nil
}

func ReadIntEnvVar(name string) (int, error) {
	valueStr, err := ReadStringEnvVar(name)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("Could not convert %s to a number: %v", name, err)
	}
	return value, nil
}

func ReadStringEnvVarOr(name string, or string) string {
	value, err := ReadStringEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func ReadIntEnvVarOr(name string, or int) int {
	value, err := ReadIntEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func readBoolEnvVarOr(name string, or bool) bool {
	valueStr, err := ReadStringEnvVar(name)
	if err != nil {
		return or
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return or
	}
	return value
}
