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
	if s.Fields[0].TSType != "string" || s.Fields[0].JSONName != "name" {
		t.Fatalf("unexpected first field TS info: TSType=%q JSONName=%q", s.Fields[0].TSType, s.Fields[0].JSONName)
	}
	if s.Fields[1].Name != "ParentID" || s.Fields[1].GoType != "int64" {
		t.Fatalf("unexpected second field: %+v", s.Fields[1])
	}
	if s.Fields[1].TSType != "number" || !s.Fields[1].Optional {
		t.Fatalf("unexpected second field TS info: TSType=%q Optional=%v", s.Fields[1].TSType, s.Fields[1].Optional)
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
	if _, err := os.Stat(filepath.Join(tsOutDir, "types.ts")); err != nil {
		t.Fatalf("expected generated types.ts: %v", err)
	}

	// types.ts should contain the EchoReq interface
	typesBody, err := os.ReadFile(filepath.Join(tsOutDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	typesStr := string(typesBody)
	if !strings.Contains(typesStr, "export interface EchoReq") {
		t.Fatalf("types.ts should contain EchoReq interface, got:\n%s", typesStr)
	}
	if !strings.Contains(typesStr, "msg: string;") {
		t.Fatalf("types.ts should contain msg field, got:\n%s", typesStr)
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
	// client.ts should import types from types.ts
	if !strings.Contains(tsStr, `import type { EchoReq } from "./types"`) {
		t.Fatalf("client.ts should import EchoReq from types, got:\n%s", tsStr)
	}
	// Method signature should use EchoReq, not unknown
	if !strings.Contains(tsStr, "reqBody: EchoReq") {
		t.Fatalf("client.ts Echo method should use EchoReq type, got:\n%s", tsStr)
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

	// client.ts should import types and use concrete type in method signature
	if !strings.Contains(tsStr, `import type { EchoReq } from "./types"`) {
		t.Fatalf("client.ts should import EchoReq, got:\n%s", tsStr)
	}
	if !strings.Contains(tsStr, "reqBody: EchoReq") {
		t.Fatalf("client.ts Echo should use EchoReq parameter type, got:\n%s", tsStr)
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

func TestTsTypeName(t *testing.T) {
	tests := []struct {
		name string
		typ  *parser.Type
		want string
	}{
		{"nil", nil, "unknown"},
		{"bool", &parser.Type{Name: "bool"}, "boolean"},
		{"i32", &parser.Type{Name: "i32"}, "number"},
		{"i64", &parser.Type{Name: "i64"}, "number"},
		{"double", &parser.Type{Name: "double"}, "number"},
		{"string", &parser.Type{Name: "string"}, "string"},
		{"binary", &parser.Type{Name: "binary"}, "string"},
		{"byte", &parser.Type{Name: "byte"}, "number"},
		{"i8", &parser.Type{Name: "i8"}, "number"},
		{"i16", &parser.Type{Name: "i16"}, "number"},
		{"void", &parser.Type{Name: "void"}, "void"},
		{"list<string>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "string"}}, "string[]"},
		{"list<i64>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "i64"}}, "number[]"},
		{"set<string>", &parser.Type{Name: "set", ValueType: &parser.Type{Name: "string"}}, "string[]"},
		{"list<EchoReq>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "EchoReq"}}, "EchoReq[]"},
		{"set<UserInfo>", &parser.Type{Name: "set", ValueType: &parser.Type{Name: "UserInfo"}}, "UserInfo[]"},
		{"map<string,i32>", &parser.Type{Name: "map", KeyType: &parser.Type{Name: "string"}, ValueType: &parser.Type{Name: "i32"}}, "Record<string, number>"},
		{"map<string,UserInfo>", &parser.Type{Name: "map", KeyType: &parser.Type{Name: "string"}, ValueType: &parser.Type{Name: "UserInfo"}}, "Record<string, UserInfo>"},
		{"map<i64,string>", &parser.Type{Name: "map", KeyType: &parser.Type{Name: "i64"}, ValueType: &parser.Type{Name: "string"}}, "Record<number, string>"},
		{"struct ref", &parser.Type{Name: "EchoReq"}, "EchoReq"},
		{"struct ref PascalCase", &parser.Type{Name: "UserInfo"}, "UserInfo"},
		{"nested list<list<string>>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "list", ValueType: &parser.Type{Name: "string"}}}, "string[][]"},
		{"list nil elem", &parser.Type{Name: "list"}, "unknown[]"},
		{"map nil key/val", &parser.Type{Name: "map"}, "Record<string, unknown>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tsTypeName(tt.typ)
			if got != tt.want {
				t.Errorf("tsTypeName(%v) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

// TestGenerateThriftArtifacts_ComplexTypes verifies generation with multiple structs,
// optional fields, list/map fields, and struct references in method signatures.
func TestGenerateThriftArtifacts_ComplexTypes(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "order", "v1", "order.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go order

struct OrderItem {
  1: string product_name
  2: i32 quantity
  3: double price
}

struct CreateOrderReq {
  1: string customer_id
  2: list<OrderItem> items
  3: optional map<string, string> metadata
}

struct OrderResp {
  1: i64 order_id
  2: string status
  3: optional list<string> tags
}

service OrderService {
  OrderResp CreateOrder(1: CreateOrderReq req)
  void CancelOrder(1: i64 id)
  list<OrderResp> ListOrders(1: string customer_id)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("GenerateThriftArtifacts: %v", err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "order", "v1")

	// types.ts validation
	typesBody, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	typesStr := string(typesBody)

	// All three structs should be present
	for _, iface := range []string{"export interface OrderItem", "export interface CreateOrderReq", "export interface OrderResp"} {
		if !strings.Contains(typesStr, iface) {
			t.Errorf("types.ts should contain %q, got:\n%s", iface, typesStr)
		}
	}

	// Field types should use TS types
	expectedFields := []string{
		"product_name: string;",
		"quantity: number;",
		"price: number;",
		"customer_id: string;",
		"items: OrderItem[];",
		"metadata?: Record<string, string>;",
		"order_id: number;",
		"status: string;",
		"tags?: string[];",
	}
	for _, f := range expectedFields {
		if !strings.Contains(typesStr, f) {
			t.Errorf("types.ts should contain %q, got:\n%s", f, typesStr)
		}
	}

	// client.ts validation
	clientBody, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	clientStr := string(clientBody)

	// Should import types used in method signatures
	if !strings.Contains(clientStr, "import type {") {
		t.Fatalf("client.ts should have import statement, got:\n%s", clientStr)
	}
	for _, typeName := range []string{"CreateOrderReq", "OrderResp"} {
		if !strings.Contains(clientStr, typeName) {
			t.Errorf("client.ts should reference %q, got:\n%s", typeName, clientStr)
		}
	}

	// Method signatures should use concrete types
	if !strings.Contains(clientStr, "reqBody: CreateOrderReq") {
		t.Errorf("CreateOrder should use CreateOrderReq param, got:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "Promise<OrderResp>") {
		t.Errorf("CreateOrder should return Promise<OrderResp>, got:\n%s", clientStr)
	}

	// Go types should also be generated
	goDir := filepath.Join(projectDir, "internal", "api", "my-app", "order", "v1")
	goBody, err := os.ReadFile(filepath.Join(goDir, "order.go"))
	if err != nil {
		t.Fatal(err)
	}
	goStr := string(goBody)
	for _, structName := range []string{"type OrderItem struct", "type CreateOrderReq struct", "type OrderResp struct"} {
		if !strings.Contains(goStr, structName) {
			t.Errorf("order.go should contain %q, got:\n%s", structName, goStr)
		}
	}
}

// TestGenerateThriftArtifacts_NoStructs verifies types.ts is NOT generated
// when the thrift file has no struct definitions.
func TestGenerateThriftArtifacts_NoStructs(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "ping", "v1", "ping.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go ping

service PingService {
  string Ping(1: string msg)
  void Health()
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("GenerateThriftArtifacts: %v", err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "ping", "v1")

	// client.ts should exist
	if _, err := os.Stat(filepath.Join(tsDir, "client.ts")); err != nil {
		t.Fatalf("expected client.ts: %v", err)
	}

	// types.ts should NOT exist
	if _, err := os.Stat(filepath.Join(tsDir, "types.ts")); err == nil {
		t.Fatal("types.ts should NOT be generated when there are no structs")
	}

	// client.ts should NOT contain import statement
	clientBody, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(clientBody), "import type") {
		t.Fatalf("client.ts should not import types when no structs, got:\n%s", clientBody)
	}
}

// TestParseThriftServiceInfo_TSImportTypes verifies TSImportTypes only contains
// struct names that are actually used in method signatures.
func TestParseThriftServiceInfo_TSImportTypes(t *testing.T) {
	dir := t.TempDir()
	thriftPath := filepath.Join(dir, "test.thrift")
	content := `namespace go test

struct ReqA {
  1: string msg
}

struct RespB {
  1: i64 id
}

struct Unused {
  1: string data
}

service TestService {
  RespB DoWork(1: ReqA req)
  string Ping(1: string msg)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseThriftServiceInfo(thriftPath)
	if err != nil {
		t.Fatalf("ParseThriftServiceInfo: %v", err)
	}

	// Should have 3 structs defined
	if len(info.Structs) != 3 {
		t.Fatalf("structs = %d, want 3", len(info.Structs))
	}

	// TSImportTypes should only include ReqA and RespB, NOT Unused
	if len(info.TSImportTypes) != 2 {
		t.Fatalf("TSImportTypes = %v, want 2 entries", info.TSImportTypes)
	}

	importMap := make(map[string]bool)
	for _, name := range info.TSImportTypes {
		importMap[name] = true
	}
	if !importMap["ReqA"] {
		t.Error("TSImportTypes should contain ReqA")
	}
	if !importMap["RespB"] {
		t.Error("TSImportTypes should contain RespB")
	}
	if importMap["Unused"] {
		t.Error("TSImportTypes should NOT contain Unused")
	}
}

// TestParseThriftServiceInfo_TSImportTypes_ContainerReturn verifies that struct types
// used only in container return types (e.g. list<T>) are included in TSImportTypes.
// Regression test for: https://github.com/castle-x/gve/issues/TS-import-incomplete
func TestParseThriftServiceInfo_TSImportTypes_ContainerReturn(t *testing.T) {
	dir := t.TempDir()
	thriftPath := filepath.Join(dir, "file.thrift")
	content := `namespace go file

struct FileTreeItem {
  1: string name
  4: optional list<FileTreeItem> children
}

struct GetTreeRequest {
  1: required string space_id
}

service FileService {
  list<FileTreeItem> GetTree(1: GetTreeRequest req)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ParseThriftServiceInfo(thriftPath)
	if err != nil {
		t.Fatalf("ParseThriftServiceInfo: %v", err)
	}

	importMap := make(map[string]bool)
	for _, name := range info.TSImportTypes {
		importMap[name] = true
	}

	// GetTreeRequest is used as parameter — must be imported
	if !importMap["GetTreeRequest"] {
		t.Errorf("TSImportTypes should contain GetTreeRequest, got %v", info.TSImportTypes)
	}
	// FileTreeItem is used in return type list<FileTreeItem> — must also be imported
	if !importMap["FileTreeItem"] {
		t.Errorf("TSImportTypes should contain FileTreeItem (from list<FileTreeItem> return), got %v", info.TSImportTypes)
	}
}

// TestCollectStructRefs verifies recursive extraction of struct names from Thrift types.
func TestCollectStructRefs(t *testing.T) {
	tests := []struct {
		name string
		typ  *parser.Type
		want []string
	}{
		{"nil", nil, nil},
		{"primitive", &parser.Type{Name: "string"}, nil},
		{"struct", &parser.Type{Name: "UserInfo"}, []string{"UserInfo"}},
		{"list<struct>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "Item"}}, []string{"Item"}},
		{"list<string>", &parser.Type{Name: "list", ValueType: &parser.Type{Name: "string"}}, nil},
		{"map<string,struct>", &parser.Type{Name: "map", KeyType: &parser.Type{Name: "string"}, ValueType: &parser.Type{Name: "Item"}}, []string{"Item"}},
		{"set<struct>", &parser.Type{Name: "set", ValueType: &parser.Type{Name: "Order"}}, []string{"Order"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectStructRefs(tt.typ)
			if len(got) != len(tt.want) {
				t.Fatalf("collectStructRefs = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("collectStructRefs[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestGenerateThriftArtifacts_TypesTSIdempotent verifies types.ts is idempotent.
func TestGenerateThriftArtifacts_TypesTSIdempotent(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "user", "v1", "user.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go user

struct GetUserReq {
  1: i64 id
}

struct UserResp {
  1: i64 id
  2: string name
  3: optional string email
}

service UserService {
  UserResp GetUser(1: GetUserReq req)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "user", "v1")

	// First run
	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	firstTypes, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	firstClient, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}

	// Second run
	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	secondTypes, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	secondClient, err := os.ReadFile(filepath.Join(tsDir, "client.ts"))
	if err != nil {
		t.Fatal(err)
	}

	if string(firstTypes) != string(secondTypes) {
		t.Fatalf("types.ts idempotency failed:\nfirst:\n%s\nsecond:\n%s", firstTypes, secondTypes)
	}
	if string(firstClient) != string(secondClient) {
		t.Fatalf("client.ts idempotency failed:\nfirst:\n%s\nsecond:\n%s", firstClient, secondClient)
	}
}

// TestGenerateThriftArtifacts_TypesTSBiomeCompliance verifies types.ts output
// is compatible with Biome linter rules.
func TestGenerateThriftArtifacts_TypesTSBiomeCompliance(t *testing.T) {
	projectDir := t.TempDir()
	thriftPath := filepath.Join(projectDir, "api", "my-app", "task", "v1", "task.thrift")
	if err := os.MkdirAll(filepath.Dir(thriftPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `namespace go task

struct TaskReq {
  1: string name
  2: optional i64 parent_id
  3: list<string> tags
}

struct TaskResp {
  1: i64 id
  2: string status
}

service TaskService {
  TaskResp CreateTask(1: TaskReq req)
}
`
	if err := os.WriteFile(thriftPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		t.Fatalf("GenerateThriftArtifacts: %v", err)
	}

	tsDir := filepath.Join(projectDir, "site", "src", "api", "my-app", "task", "v1")
	typesBody, err := os.ReadFile(filepath.Join(tsDir, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	typesStr := string(typesBody)

	// Must use semicolons
	if strings.Contains(typesStr, "name: string\n") {
		t.Fatalf("types.ts should use semicolons, got:\n%s", typesStr)
	}

	// No single-quoted strings
	if strings.Contains(typesStr, "'") {
		t.Fatalf("types.ts should not contain single quotes, got:\n%s", typesStr)
	}

	// export interface keyword (not type alias)
	if !strings.Contains(typesStr, "export interface TaskReq") {
		t.Fatalf("types.ts should use 'export interface', got:\n%s", typesStr)
	}

	// Optional fields should use ? syntax
	if !strings.Contains(typesStr, "parent_id?:") {
		t.Fatalf("types.ts optional field should use ?, got:\n%s", typesStr)
	}

	// Required fields should NOT have ?
	if strings.Contains(typesStr, "name?:") {
		t.Fatalf("types.ts required field should not have ?, got:\n%s", typesStr)
	}
}
