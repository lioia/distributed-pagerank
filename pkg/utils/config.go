package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	C         float64
	Threshold float64
	Graph     string
	Output    string
}

// Load config.json (C, Threshold and graph file)
func LoadConfiguration() (config Config, err error) {
	// Try to open the config.json file
	if _, err = os.Open("config.json"); err != nil {
		err = fmt.Errorf("file does not exists: %v", err)
		log.Printf("file does not exists: %v", err)
		return
	}
	// File exists -> load configuration
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		err = fmt.Errorf("read: %v", err)
		return
	}
	// Parse config.json into Config struct
	if err = json.Unmarshal(bytes, &config); err != nil {
		err = fmt.Errorf("parse: %v", err)
		return
	}
	return
}
