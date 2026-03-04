package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseThriftServiceInfo(t *testing.T) {
	dir := t.TempDir()
	thriftPath := filepath.Join(dir, "task.thrift")
	content := `namespace go task

service TaskService {
  i64 GetTask(1: i64 id)
  void Ping()
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseThriftServiceInfo(thriftPath)
	if err != nil {
		t.Fatalf("ParseThriftServiceInfo: %v", err)
	}

	if info.PackageName != "task" {
		t.Fatalf("package = %q, want task", info.PackageName)
	}
	if info.ServiceName != "TaskService" {
		t.Fatalf("service = %q, want TaskService", info.ServiceName)
	}
	if len(info.Methods) != 2 {
		t.Fatalf("methods = %d, want 2", len(info.Methods))
	}
	if info.Methods[0].Name != "GetTask" || info.Methods[0].RequestTypeTS != "number" {
		t.Fatalf("unexpected first method: %+v", info.Methods[0])
	}
	if info.Methods[1].ReturnTypeTS != "void" {
		t.Fatalf("expected void return in second method, got %+v", info.Methods[1])
	}
}

func TestGenerateThriftArtifacts(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go task

struct EchoReq {
  1: string msg
}

service TaskService {
  string Echo(1: EchoReq req)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("GenerateThriftArtifacts: %v", err)
	}

	goOutDir := filepath.Join(projectDir, "internal", "api", "my-app", "task", "v1")
	tsOutDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "task", "v1")

	for _, f := range []string{"task.go", "client.go"} {
		if _, err := os.Stat(filepath.Join(goOutDir, f)); err != nil {
			t.Fatalf("expected generated %s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(tsOutDir, "client.ts")); err != nil {
		t.Fatalf("expected generated client.ts: %v", err)
	}

	goBody, err := os.ReadFile(filepath.Join(goOutDir, "client.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goBody), "func (c *TaskServiceClient) Echo") {
		t.Fatalf("client.go should contain Echo method, got:\n%s", goBody)
	}

	tsBody, err := os.ReadFile(filepath.Join(tsOutDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tsBody), "async Echo") {
		t.Fatalf("client.ts should contain Echo method, got:\n%s", tsBody)
	}

	entries, err := os.ReadDir(filepath.Dir(thriftPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "task.thrift" {
		t.Fatalf("expected api dir to contain only task.thrift, got %+v", entries)
	}
}
