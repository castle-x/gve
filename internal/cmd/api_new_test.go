package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAPINew_DefaultVersion(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPINew(nil, []string{"my-app/user"}); err != nil {
		t.Fatalf("runAPINew: %v", err)
	}

	thriftPath := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	body, err := os.ReadFile(thriftPath)
	if err != nil {
		t.Fatalf("read generated thrift: %v", err)
	}
	content := string(body)
	for _, s := range []string{"namespace go user", "namespace js user", "service UserService"} {
		if !strings.Contains(content, s) {
			t.Errorf("generated thrift should contain %q, got:\n%s", s, content)
		}
	}
}

func TestRunAPINew_WithVersion(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPINew(nil, []string{"my-app/task", "v2"}); err != nil {
		t.Fatalf("runAPINew: %v", err)
	}

	thriftPath := filepath.Join(projectDir, "api", "my-app", "task", "v2", "task.thrift")
	if _, err := os.Stat(thriftPath); err != nil {
		t.Fatalf("generated thrift not found: %v", err)
	}
}

func TestRunAPINew_InvalidTarget(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	err := runAPINew(nil, []string{"invalid"})
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
	if !strings.Contains(err.Error(), "project/resource") {
		t.Fatalf("expected format error, got: %v", err)
	}
}

func TestRunAPINew_FileExists(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	thriftPath := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(thriftPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	err := runAPINew(nil, []string{"my-app/user"})
	if err == nil {
		t.Fatal("expected file exists error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		t.Fatalf("expected already exists error, got: %v", err)
	}
}
