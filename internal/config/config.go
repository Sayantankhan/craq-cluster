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
	Nodes []NodeInfo `json:"nodes"`
	DB    DBInfo     `json:"db"`
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

func (c *Config) FindNodeAndSuccessor(id string) (*NodeInfo, *NodeInfo) {
	for i, n := range c.Nodes {
		if n.ID == id {
			var next *NodeInfo
			if i+1 < len(c.Nodes) {
				next = &c.Nodes[i+1]
			}
			return &n, next
		}
	}
	return nil, nil
}
