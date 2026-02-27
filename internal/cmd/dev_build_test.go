package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractProjectName_FromGoMod(t *testing.T) {
	dir := t.TempDir()

	goModContent := "module github.com/castle-x/my-awesome-project\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	name, err := extractProjectName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "my-awesome-project" {
		t.Errorf("got %q, want %q", name, "my-awesome-project")
	}
}

func TestExtractProjectName_SimpleModule(t *testing.T) {
	dir := t.TempDir()

	goModContent := "module myapp\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	name, err := extractProjectName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "myapp" {
		t.Errorf("got %q, want %q", name, "myapp")
	}
}

func TestExtractProjectName_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	name, err := extractProjectName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != filepath.Base(dir) {
		t.Errorf("got %q, want %q", name, filepath.Base(dir))
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{15728640, "15.0 MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestBuildCrossCompileEnv(t *testing.T) {
	tests := []struct {
		os, arch string
		wantLen  int
	}{
		{"", "", 0},
		{"linux", "", 1},
		{"", "arm64", 1},
		{"darwin", "amd64", 2},
	}
	for _, tt := range tests {
		env := buildCrossCompileEnv(tt.os, tt.arch)
		if len(env) != tt.wantLen {
			t.Errorf("buildCrossCompileEnv(%q, %q) returned %d env vars, want %d", tt.os, tt.arch, len(env), tt.wantLen)
		}
	}
}
