package main

import (
	"encoding/json"
	"os"
)

type LBServer struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Health  string `json:"health"`
	Enable  bool   `json:"enable"`
}

func loadLBs(filename string) ([]LBServer, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var lbs []LBServer
	if err := json.Unmarshal(data, &lbs); err != nil {
		return nil, err
	}

	return lbs, nil
}
