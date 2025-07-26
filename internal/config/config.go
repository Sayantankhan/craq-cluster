package config

import (
	"encoding/json"
	"os"
)

type NodeInfo struct {
	ID     string `json:"id"`
	Addr   string `json:"addr"`
	IsHead bool   `json:"isHead,omitempty"`
	IsTail bool   `json:"isTail,omitempty"`
}

type DBInfo struct {
	Addr string `json:"addr"`
}

type Config struct {
	Manager string `json:"manager"`
	DB      DBInfo `json:"db"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
