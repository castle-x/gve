package asset

import (
	"encoding/json"
	"os"
)

type Meta struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Dest    string   `json:"dest,omitempty"`
	Deps    []string `json:"deps,omitempty"`
	Files   []string `json:"files"`
}

func LoadMeta(path string) (*Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
