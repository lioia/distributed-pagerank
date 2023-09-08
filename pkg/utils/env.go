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
	master, err := readStringEnvVar("MASTER")
	FailOnError("Failed to read environment variables", err)
	host := readStringEnvVarOr("HOST", "")
	port, err := readIntEnvVar("PORT")
	FailOnError("Failed to read environment variables", err)
	rabbitHost, err := readStringEnvVar("RABBIT_HOST")
	FailOnError("Failed to read environment variables", err)
	rabbitUser := readStringEnvVarOr("RABBIT_USER", "guest")
	rabbitPass := readStringEnvVarOr("RABBIT_PASSWORD", "guest")
	workQueue := readStringEnvVarOr("WORK_QUEUE", "work")
	resultQueue := readStringEnvVarOr("RESULT_QUEUE", "result")
	nodeLog := readBoolEnvVarOr("NODE_LOG", false)
	serverLog := readBoolEnvVarOr("SERVER_LOG", false)
	return EnvVars{
		Master: master, Host: host, Port: port,
		RabbitHost: rabbitHost, RabbitUser: rabbitUser, RabbitPass: rabbitPass,
		WorkQueue: workQueue, ResultQueue: resultQueue,
		NodeLog: nodeLog, ServerLog: serverLog,
	}
}

func readStringEnvVar(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s not set", name)
	}
	return value, nil
}

func readIntEnvVar(name string) (int, error) {
	valueStr, err := readStringEnvVar(name)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("Could not convert %s to a number: %v", name, err)
	}
	return value, nil
}

func readStringEnvVarOr(name string, or string) string {
	value, err := readStringEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func ReadIntEnvVarOr(name string, or int) int {
	value, err := readIntEnvVar(name)
	if err != nil {
		value = or
	}
	return value
}

func readBoolEnvVarOr(name string, or bool) bool {
	valueStr, err := readStringEnvVar(name)
	if err != nil {
		return or
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return or
	}
	return value
}
