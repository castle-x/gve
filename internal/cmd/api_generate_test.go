package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAPIGenerate_FailFast(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	first := filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift")
	second := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	for _, p := range []string{first, second} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("namespace go demo\nservice DemoService {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, projectDir)

	orig := generateThriftArtifacts
	defer func() { generateThriftArtifacts = orig }()

	called := 0
	generateThriftArtifacts = func(projectDir, thriftPath string) error {
		called++
		return errors.New("boom")
	}

	err := runAPIGenerate(nil, nil)
	if err == nil {
		t.Fatal("expected generate failure")
	}
	if called != 1 {
		t.Fatalf("expected fail-fast with one call, got %d", called)
	}
}

func TestRunAPIGenerate_ScansOnlyCanonicalLayout(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	okPath := filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift")
	ignoredPath := filepath.Join(projectDir, "api", "misc", "adhoc.thrift")
	for _, p := range []string{okPath, ignoredPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("namespace go demo\nservice DemoService {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, projectDir)

	orig := generateThriftArtifacts
	defer func() { generateThriftArtifacts = orig }()

	var seen []string
	generateThriftArtifacts = func(projectDir, thriftPath string) error {
		seen = append(seen, filepath.Clean(thriftPath))
		return nil
	}

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	if len(seen) != 1 {
		t.Fatalf("expected 1 canonical thrift file, got %d", len(seen))
	}
	if seen[0] != filepath.Clean(okPath) {
		t.Fatalf("unexpected generated path: %s", seen[0])
	}
}

func TestRunAPIGenerate_EndToEndSingleFile(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	thriftPath := filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	thrift := `namespace go task

struct EchoReq {
  1: string msg
}

service TaskService {
  string Echo(1: EchoReq req)
}
`
	if err := os.WriteFile(thriftPath, []byte(thrift), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	for _, file := range []string{"task.go", "client.go"} {
		path := filepath.Join(projectDir, "internal", "api", "my-app", "task", "v1", file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}
	tsClient := filepath.Join(projectDir, "site", "src", "api", "my-app", "task", "v1", "client.ts")
	if _, err := os.Stat(tsClient); err != nil {
		t.Fatalf("expected generated file client.ts: %v", err)
	}
	for _, file := range []string{"task.go", "client.go", "client.ts"} {
		path := filepath.Join(projectDir, "api", "my-app", "task", "v1", file)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("did not expect generated file in api root: %s", path)
		}
	}
}

func TestRunAPIGenerate_KeepThriftOnlyInRootAPI(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	thriftPath := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	thrift := `namespace go user
service UserService {
  string Ping(1: string in)
}
`
	if err := os.WriteFile(thriftPath, []byte(thrift), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	if _, err := os.Stat(thriftPath); err != nil {
		t.Fatalf("expected thrift file to remain: %v", err)
	}
	apiDir := filepath.Dir(thriftPath)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "user.thrift" {
		t.Fatalf("expected api dir to contain only user.thrift, got %+v", entries)
	}
	for _, file := range []string{"user.go", "client.go"} {
		path := filepath.Join(projectDir, "internal", "api", "my-app", "user", "v1", file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}
	if _, err := os.Stat(filepath.Join(projectDir, "site", "src", "api", "my-app", "user", "v1", "client.ts")); err != nil {
		t.Fatalf("expected generated TS client: %v", err)
	}
}
