package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadMeta_ScaffoldFields(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{
		"name": "default",
		"version": "1.0.0",
		"category": "scaffold",
		"description": "Default scaffold with dashboard layout.",
		"dest": "site",
		"deps": [],
		"files": ["package.json", "index.html", "src/App.tsx"],
		"defaultAssets": ["ui/theme-provider", "components/settings-dropdown"],
		"shadcnDeps": ["button", "card", "dialog", "sidebar"]
	}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if len(m.DefaultAssets) != 2 {
		t.Errorf("DefaultAssets len = %d, want 2", len(m.DefaultAssets))
	}
	if m.DefaultAssets[0] != "ui/theme-provider" {
		t.Errorf("DefaultAssets[0] = %q, want %q", m.DefaultAssets[0], "ui/theme-provider")
	}
	if len(m.ShadcnDeps) != 4 {
		t.Errorf("ShadcnDeps len = %d, want 4", len(m.ShadcnDeps))
	}
	if m.ShadcnDeps[0] != "button" {
		t.Errorf("ShadcnDeps[0] = %q, want %q", m.ShadcnDeps[0], "button")
	}
}

func TestLoadMeta_ScaffoldFieldsOmitted(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{"name":"button","version":"1.0.0","files":["button.tsx"]}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if m.DefaultAssets != nil {
		t.Errorf("DefaultAssets should be nil, got %v", m.DefaultAssets)
	}
	if m.ShadcnDeps != nil {
		t.Errorf("ShadcnDeps should be nil, got %v", m.ShadcnDeps)
	}
}

func TestMeta_MarshalRoundTrip(t *testing.T) {
	m := Meta{
		Schema:        "https://gve.dev/schema/meta.json",
		Name:          "spinner",
		Version:       "1.0.0",
		Category:      "ui",
		Description:   "Loading spinner.",
		Deps:          []string{},
		PeerDeps:      []string{},
		Files:         []string{"spinner.tsx"},
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

func TestMeta_MarshalRoundTrip_WithScaffoldFields(t *testing.T) {
	m := Meta{
		Name:          "default",
		Version:       "1.0.0",
		Category:      "scaffold",
		Files:         []string{"package.json"},
		DefaultAssets: []string{"ui/theme-provider"},
		ShadcnDeps:    []string{"button", "card"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var m2 Meta
	json.Unmarshal(data, &m2)
	if len(m2.DefaultAssets) != 1 || m2.DefaultAssets[0] != "ui/theme-provider" {
		t.Errorf("DefaultAssets roundtrip failed: %v", m2.DefaultAssets)
	}
	if len(m2.ShadcnDeps) != 2 || m2.ShadcnDeps[0] != "button" {
		t.Errorf("ShadcnDeps roundtrip failed: %v", m2.ShadcnDeps)
	}
}

func TestMeta_OmitEmptyScaffoldFields(t *testing.T) {
	m := Meta{
		Name:    "button",
		Version: "1.0.0",
		Files:   []string{"button.tsx"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "defaultAssets") {
		t.Error("defaultAssets should be omitted when empty")
	}
	if strings.Contains(s, "shadcnDeps") {
		t.Error("shadcnDeps should be omitted when empty")
	}
}
