package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestRunAPIGenerate_EndToEnd_TypesTS verifies the full CLI flow generates
// types.ts with correct interfaces and client.ts imports them.
func TestRunAPIGenerate_EndToEnd_TypesTS(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	thriftPath := filepath.Join(projectDir, "api", "my-app", "order", "v1", "order.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	thrift := `namespace go order

struct OrderReq {
  1: string customer_id
  2: list<string> item_ids
  3: optional string note
}

struct OrderResp {
  1: i64 order_id
  2: string status
}

service OrderService {
  OrderResp PlaceOrder(1: OrderReq req)
  void CancelOrder(1: i64 id)
}
`
	if err := os.WriteFile(thriftPath, []byte(thrift), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "order", "v1")

	// Verify all expected files exist
	for _, f := range []string{"client.ts", "types.ts"} {
		if _, err := os.Stat(filepath.Join(tsDir, f)); err != nil {
			t.Fatalf("expected generated %s: %v", f, err)
		}
	}

	// types.ts should contain both interfaces with correct fields
	typesBody, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	typesStr := string(typesBody)

	for _, expected := range []string{
		"export interface OrderReq",
		"customer_id: string;",
		"item_ids: string[];",
		"note?: string;",
		"export interface OrderResp",
		"order_id: number;",
		"status: string;",
	} {
		if !strings.Contains(typesStr, expected) {
			t.Errorf("types.ts should contain %q, got:\n%s", expected, typesStr)
		}
	}

	// client.ts should import the types used in signatures
	clientBody, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	clientStr := string(clientBody)

	if !strings.Contains(clientStr, `import type {`) {
		t.Errorf("client.ts should have type import, got:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "OrderReq") {
		t.Errorf("client.ts should reference OrderReq, got:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "OrderResp") {
		t.Errorf("client.ts should reference OrderResp, got:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, `from "./types"`) {
		t.Errorf("client.ts should import from ./types, got:\n%s", clientStr)
	}

	// Method signatures should use concrete types
	if !strings.Contains(clientStr, "reqBody: OrderReq") {
		t.Errorf("PlaceOrder should use OrderReq param, got:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "Promise<OrderResp>") {
		t.Errorf("PlaceOrder should return Promise<OrderResp>, got:\n%s", clientStr)
	}

	// Go files should also be correctly generated
	goDir := filepath.Join(projectDir, "internal", "api", "my-app", "order", "v1")
	for _, f := range []string{"order.go", "client.go"} {
		if _, err := os.Stat(filepath.Join(goDir, f)); err != nil {
			t.Fatalf("expected generated %s: %v", f, err)
		}
	}
}

// TestRunAPIGenerate_EndToEnd_MultipleFiles verifies generating from multiple
// thrift files in a single project produces independent types.ts per service.
func TestRunAPIGenerate_EndToEnd_MultipleFiles(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift"): `namespace go user
struct UserReq { 1: i64 id }
struct UserResp { 1: string name }
service UserService { UserResp GetUser(1: UserReq req) }
`,
		filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift"): `namespace go task
struct TaskReq { 1: string title }
service TaskService { string CreateTask(1: TaskReq req) }
`,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, projectDir)

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	// Both services should have types.ts
	for _, svc := range []struct {
		dir    string
		ifaces []string
	}{
		{"my-app/user/v1", []string{"UserReq", "UserResp"}},
		{"my-app/task/v1", []string{"TaskReq"}},
	} {
		tsDir := filepath.Join(projectDir, "site", "src", "api", svc.dir)
		typesBody, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
		if err != nil {
			t.Fatalf("types.ts missing for %s: %v", svc.dir, err)
		}
		typesStr := string(typesBody)
		for _, iface := range svc.ifaces {
			if !strings.Contains(typesStr, "export interface "+iface) {
				t.Errorf("%s/types.ts should contain interface %s, got:\n%s", svc.dir, iface, typesStr)
			}
		}

		clientBody, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
		if err != nil {
			t.Fatalf("client.ts missing for %s: %v", svc.dir, err)
		}
		if !strings.Contains(string(clientBody), `from "./types"`) {
			t.Errorf("%s/client.ts should import from ./types, got:\n%s", svc.dir, clientBody)
		}
	}
}

// TestRunAPIGenerate_NoStructs_NoTypesTS verifies that when a thrift file has no
// structs, types.ts is not generated and client.ts has no import.
func TestRunAPIGenerate_NoStructs_NoTypesTS(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "gve.lock"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	thriftPath := filepath.Join(projectDir, "api", "my-app", "health", "v1", "health.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(thriftPath, []byte(`namespace go health
service HealthService { string Ping(1: string msg) }
`), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	if err := runAPIGenerate(nil, nil); err != nil {
		t.Fatalf("runAPIGenerate: %v", err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "health", "v1")
	if _, err := os.Stat(filepath.Join(tsDir, "types.ts")); err == nil {
		t.Fatal("types.ts should not exist when there are no structs")
	}
	clientBody, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(clientBody), "import type") {
		t.Fatalf("client.ts should not have import when no structs, got:\n%s", clientBody)
	}
}
