package asset

import (
	"encoding/json"
	"os"
	"strings"
)

type Meta struct {
	Schema      string   `json:"$schema,omitempty"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Category    string   `json:"category,omitempty"`
	Description string   `json:"description,omitempty"`
	Dest        string   `json:"dest,omitempty"`
	Deps        []string `json:"deps,omitempty"`
	PeerDeps    []string `json:"peerDeps,omitempty"`
	Files       []string `json:"files"`
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

// InferCategory derives the asset category from its registry path.
// e.g. "ui/spinner/v1.0.0" -> "ui", "components/data-table/v2.0.0" -> "component"
func InferCategory(assetPath string) string {
	if assetPath == "" {
		return ""
	}
	parts := strings.SplitN(assetPath, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "scaffold":
		return "scaffold"
	case "ui":
		return "ui"
	case "components":
		return "component"
	case "global":
		return "global"
	default:
		return ""
	}
}
