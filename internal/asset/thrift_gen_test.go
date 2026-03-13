package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/thriftgo/parser"
)

func TestParseThriftServiceInfo(t *testing.T) {
	dir := t.TempDir()
	thriftPath := filepath.Join(dir, "task.thrift")
	content := `namespace go task

struct TaskReq {
  1: string name
  2: optional i64 parent_id
}

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

	// Verify struct extraction
	if len(info.Structs) != 1 {
		t.Fatalf("structs = %d, want 1", len(info.Structs))
	}
	s := info.Structs[0]
	if s.Name != "TaskReq" {
		t.Fatalf("struct name = %q, want TaskReq", s.Name)
	}
	if len(s.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(s.Fields))
	}
	if s.Fields[0].Name != "Name" || s.Fields[0].GoType != "string" {
		t.Fatalf("unexpected first field: %+v", s.Fields[0])
	}
	if s.Fields[1].Name != "ParentID" || s.Fields[1].GoType != "int64" {
		t.Fatalf("unexpected second field: %+v", s.Fields[1])
	}
	if !strings.Contains(s.Fields[1].JSONTag, "omitempty") {
		t.Fatalf("expected omitempty on optional field, got %s", s.Fields[1].JSONTag)
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

	// task.go should contain our struct definition, not thriftgo output
	taskBody, err := os.ReadFile(filepath.Join(goOutDir, "task.go"))
	if err != nil {
		t.Fatal(err)
	}
	taskStr := string(taskBody)
	if !strings.Contains(taskStr, "type EchoReq struct") {
		t.Fatalf("task.go should contain EchoReq struct definition, got:\n%s", taskStr)
	}
	if !strings.Contains(taskStr, "package task") {
		t.Fatalf("task.go should have package task, got:\n%s", taskStr)
	}
	// Should NOT contain thriftgo artifacts
	if strings.Contains(taskStr, "TProtocol") || strings.Contains(taskStr, "ReadStructBegin") {
		t.Fatalf("task.go should not contain thriftgo serialization code, got:\n%s", taskStr)
	}

	// client.go should use HTTPClient naming
	goBody, err := os.ReadFile(filepath.Join(goOutDir, "client.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goBody), "func (c *TaskServiceHTTPClient) Echo") {
		t.Fatalf("client.go should contain TaskServiceHTTPClient.Echo method, got:\n%s", goBody)
	}

	tsBody, err := os.ReadFile(filepath.Join(tsOutDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	tsStr := string(tsBody)
	if !strings.Contains(tsStr, "async Echo") {
		t.Fatalf("client.ts should contain Echo method, got:\n%s", tsStr)
	}
	// Regression: parameter property syntax is incompatible with erasableSyntaxOnly
	if strings.Contains(tsStr, "constructor(private") {
		t.Fatalf("client.ts must not use parameter property syntax (incompatible with erasableSyntaxOnly), got:\n%s", tsStr)
	}
	// ClientOptions interface must be present
	if !strings.Contains(tsStr, "interface ClientOptions") {
		t.Fatalf("client.ts should contain ClientOptions interface, got:\n%s", tsStr)
	}
	// Constructor should accept optional ClientOptions
	if !strings.Contains(tsStr, "options?: ClientOptions") {
		t.Fatalf("client.ts constructor should accept options?: ClientOptions, got:\n%s", tsStr)
	}
	// Methods should use injected fetch
	if !strings.Contains(tsStr, "this.options.fetch ?? globalThis.fetch") {
		t.Fatalf("client.ts should use this.options.fetch ?? globalThis.fetch, got:\n%s", tsStr)
	}
	// Methods should spread baseHeaders
	if !strings.Contains(tsStr, "...this.options.baseHeaders") {
		t.Fatalf("client.ts should spread baseHeaders, got:\n%s", tsStr)
	}
	// Methods should call onError before throw
	if !strings.Contains(tsStr, `this.options.onError?.(error, "Echo")`) {
		t.Fatalf("client.ts should call onError with method name, got:\n%s", tsStr)
	}

	entries, err := os.ReadDir(filepath.Dir(thriftPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "task.thrift" {
		t.Fatalf("expected api dir to contain only task.thrift, got %+v", entries)
	}
}

func TestGenerateThriftArtifacts_IdempotentNoStaleSubdir(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go user

struct GetUserReq {
  1: i64 id
}

service UserService {
  string GetUser(1: GetUserReq req)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	goOutDir := filepath.Join(projectDir, "internal", "api", "my-app", "user", "v1")

	// First run
	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(goOutDir, "user.go"))
	if err != nil {
		t.Fatalf("user.go missing after first run: %v", err)
	}

	// Second run (idempotent)
	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(goOutDir, "user.go"))
	if err != nil {
		t.Fatalf("user.go missing after second run: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("idempotency failed: first and second run produced different output")
	}
}

func TestGenerateThriftArtifacts_TSBiomeCompliance(t *testing.T) {
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
  void Ping()
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("GenerateThriftArtifacts: %v", err)
	}

	tsBody, err := os.ReadFile(filepath.Join(projectDir, "site", "src", "api", "my-app", "task", "v1", "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	tsStr := string(tsBody)

	// Semicolons: statements must end with ;
	if strings.Contains(tsStr, "private readonly baseUrl: string\n") {
		t.Fatalf("client.ts should use semicolons, got:\n%s", tsStr)
	}

	// Double quotes: no single-quoted strings
	if strings.Contains(tsStr, "'POST'") || strings.Contains(tsStr, "'Content-Type'") {
		t.Fatalf("client.ts should use double quotes, got:\n%s", tsStr)
	}

	// void method must not use "as void" (noConfusingVoidType)
	if strings.Contains(tsStr, "as void") {
		t.Fatalf("client.ts must not cast to void (noConfusingVoidType), got:\n%s", tsStr)
	}

	// void method must still consume the response body
	if !strings.Contains(tsStr, "async Ping") {
		t.Fatalf("client.ts should contain Ping method, got:\n%s", tsStr)
	}

	// No trailing blank line before closing brace
	if strings.HasSuffix(strings.TrimRight(tsStr, "\n"), "\n\n}") {
		t.Fatalf("client.ts should not have trailing blank line before closing brace, got:\n%s", tsStr)
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_name", "UserName"},
		{"id", "ID"},
		{"user_id", "UserID"},
		{"parent_url", "ParentURL"},
		{"http_status", "HTTPStatus"},
		{"created_at", "CreatedAt"},
		{"api_key", "APIKey"},
		{"simple", "Simple"},
		{"cpu_usage", "CPUUsage"},
	}
	for _, tt := range tests {
		got := toPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestThriftTypeToGo(t *testing.T) {
	tests := []struct {
		name string
		typ  *parser.Type
		want string
	}{
		{"nil", nil, "any"},
		{"bool", &parser.Type{Name: "bool"}, "bool"},
		{"i32", &parser.Type{Name: "i32"}, "int32"},
		{"i64", &parser.Type{Name: "i64"}, "int64"},
		{"double", &parser.Type{Name: "double"}, "float64"},
		{"string", &parser.Type{Name: "string"}, "string"},
		{"binary", &parser.Type{Name: "binary"}, "[]byte"},
		{"byte", &parser.Type{Name: "byte"}, "int8"},
		{"i8", &parser.Type{Name: "i8"}, "int8"},
		{"i16", &parser.Type{Name: "i16"}, "int16"},
		{"list<string>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "string"}}, "[]string"},
		{"set<i64>", &parser.Type{Name: "set", ValueType: &parser.Type{Name: "i64"}}, "[]int64"},
		{"map<string,i32>", &parser.Type{Name: "map", KeyType: &parser.Type{Name: "string"}, ValueType: &parser.Type{Name: "i32"}}, "map[string]int32"},
		{"struct ref", &parser.Type{Name: "UserInfo"}, "UserInfo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := thriftTypeToGo(tt.typ)
			if got != tt.want {
				t.Errorf("thriftTypeToGo(%v) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestBuildJSONTag(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		optional bool
		want     string
	}{
		{"required", "user_name", false, "`json:\"user_name\"`"},
		{"optional", "avatar", true, "`json:\"avatar,omitempty\"`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildJSONTag(tt.field, tt.optional)
			if got != tt.want {
				t.Errorf("buildJSONTag(%q, %v) = %q, want %q", tt.field, tt.optional, got, tt.want)
			}
		})
	}
}
