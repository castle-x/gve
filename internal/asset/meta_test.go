package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMeta_V2Fields(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{
		"$schema": "https://gve.dev/schema/meta.json",
		"name": "data-table",
		"version": "2.0.0",
		"category": "component",
		"description": "A data table with sorting and pagination.",
		"dest": "",
		"deps": ["@tanstack/react-table"],
		"peerDeps": ["ui/button", "ui/spinner"],
		"files": ["data-table.tsx"]
	}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if m.Category != "component" {
		t.Errorf("Category = %q, want %q", m.Category, "component")
	}
	if m.Description != "A data table with sorting and pagination." {
		t.Errorf("Description = %q", m.Description)
	}
	if len(m.PeerDeps) != 2 || m.PeerDeps[0] != "ui/button" {
		t.Errorf("PeerDeps = %v", m.PeerDeps)
	}
}

func TestLoadMeta_V1Compat(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{"name":"button","version":"1.0.0","deps":[],"files":["button.tsx"]}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if m.Category != "" {
		t.Errorf("Category should be empty for v1, got %q", m.Category)
	}
	if m.PeerDeps != nil {
		t.Errorf("PeerDeps should be nil for v1, got %v", m.PeerDeps)
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"scaffold/default/v1.0.0", "scaffold"},
		{"ui/spinner/v1.0.0", "ui"},
		{"components/data-table/v2.0.0", "component"},
		{"global/theme/v1.0.0", "global"},
		{"assets/button/v1.0.0", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferCategory(tt.path)
			if got != tt.want {
				t.Errorf("InferCategory(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestMeta_MarshalRoundTrip(t *testing.T) {
	m := Meta{
		Schema:      "https://gve.dev/schema/meta.json",
		Name:        "spinner",
		Version:     "1.0.0",
		Category:    "ui",
		Description: "Loading spinner.",
		Deps:        []string{},
		PeerDeps:    []string{},
		Files:       []string{"spinner.tsx"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var m2 Meta
	json.Unmarshal(data, &m2)
	if m2.Schema != m.Schema || m2.Category != m.Category {
		t.Errorf("roundtrip mismatch: %+v", m2)
	}
}
