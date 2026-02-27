package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffold(t *testing.T) {
	dir := t.TempDir()
	data := ScaffoldData{
		ProjectName: "my-app",
		UIRegistry:  "github.com/castle-x/wk-ui",
		APIRegistry: "github.com/castle-x/wk-api",
	}

	err := Scaffold(dir, data)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	// Check all template files exist and have expected content
	tests := []struct {
		path       string
		contains   []string
		exactMatch string
	}{
		{"go.mod", []string{"module my-app", "go 1."}, ""},
		{"Makefile", []string{"my-app", "build-web", "build-backend"}, ""},
		{".gitignore", []string{"dist/", ".gve/", "site/node_modules"}, ""},
		{"gve.lock", []string{`"version": "1"`, "wk-ui", "wk-api", `"assets": {}`}, ""},
		{"cmd/server/main.go", []string{"my-app", "site.", "ListenAndServe", "/api/health"}, ""},
	}

	for _, tt := range tests {
		fullPath := filepath.Join(dir, tt.path)
		body, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("read %s: %v", tt.path, err)
			continue
		}
		content := string(body)
		for _, sub := range tt.contains {
			if !strings.Contains(content, sub) {
				t.Errorf("%s: expected to contain %q, got:\n%s", tt.path, sub, content)
			}
		}
		if tt.exactMatch != "" && content != tt.exactMatch {
			t.Errorf("%s: content mismatch", tt.path)
		}
	}

	// Check placeholder dirs with .gitkeep
	keepDirs := []string{"internal/handler", "internal/service", "api"}
	for _, d := range keepDirs {
		gitkeep := filepath.Join(dir, d, ".gitkeep")
		if _, err := os.Stat(gitkeep); err != nil {
			t.Errorf("missing %s: %v", gitkeep, err)
		}
	}
}

func TestScaffold_ProjectNameInjected(t *testing.T) {
	dir := t.TempDir()
	data := ScaffoldData{
		ProjectName: "custom-project",
		UIRegistry:  "https://example.com/ui",
		APIRegistry: "https://example.com/api",
	}

	err := Scaffold(dir, data)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	goMod := filepath.Join(dir, "go.mod")
	body, _ := os.ReadFile(goMod)
	if !strings.Contains(string(body), "module custom-project") {
		t.Errorf("go.mod should contain module custom-project, got %s", body)
	}

	lockPath := filepath.Join(dir, "gve.lock")
	body, _ = os.ReadFile(lockPath)
	content := string(body)
	if !strings.Contains(content, "https://example.com/ui") || !strings.Contains(content, "https://example.com/api") {
		t.Errorf("gve.lock should contain registry URLs, got %s", content)
	}
}
