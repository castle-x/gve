package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	gtmpl "github.com/castle-x/gve/internal/template"
	"github.com/cloudwego/thriftgo/parser"
)

var parseThriftFile = parser.ParseFile

type ServiceMethod struct {
	Name          string
	RequestType   string
	ReturnType    string
	RequestTypeTS string
	ReturnTypeTS  string
}

type StructField struct {
	Name    string // PascalCase Go field name
	GoType  string // Go type
	JSONTag string // full JSON tag with backticks
}

type StructDef struct {
	Name   string
	Fields []StructField
}

type ThriftServiceInfo struct {
	PackageName string
	ServiceName string
	Methods     []ServiceMethod
	Structs     []StructDef
}

func GenerateThriftArtifacts(projectDir, thriftPath string) error {
	absPath, err := filepath.Abs(thriftPath)
	if err != nil {
		return fmt.Errorf("resolve thrift path: %w", err)
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolve project path: %w", err)
	}

	info, err := ParseThriftServiceInfo(absPath)
	if err != nil {
		return err
	}

	apiRoot := filepath.Join(absProjectDir, "api")
	relThriftPath, err := filepath.Rel(apiRoot, absPath)
	if err != nil {
		return fmt.Errorf("resolve thrift relative path: %w", err)
	}
	if strings.HasPrefix(relThriftPath, "..") {
		return fmt.Errorf("thrift path %s is outside project api dir", absPath)
	}
	relDir := filepath.Dir(relThriftPath)

	goDir := filepath.Join(absProjectDir, "internal", "api", relDir)
	tsDir := filepath.Join(absProjectDir, "site", "src", "api", relDir)
	if err := os.MkdirAll(goDir, 0755); err != nil {
		return fmt.Errorf("create go output dir: %w", err)
	}
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		return fmt.Errorf("create ts output dir: %w", err)
	}

	// Render Go struct definitions from Thrift AST (replaces thriftgo codegen)
	baseName := strings.TrimSuffix(filepath.Base(absPath), ".thrift")
	typesBody, err := gtmpl.RenderFileTemplate("api_types_go.tmpl", info)
	if err != nil {
		return fmt.Errorf("render types template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(goDir, baseName+".go"), typesBody, 0644); err != nil {
		return fmt.Errorf("write %s.go: %w", baseName, err)
	}

	if err := writeGeneratedClientFiles(goDir, tsDir, info); err != nil {
		return err
	}

	return nil
}

func ParseThriftServiceInfo(thriftPath string) (*ThriftServiceInfo, error) {
	ast, err := parseThriftFile(thriftPath, nil, false)
	if err != nil {
		return nil, fmt.Errorf("parse thrift %s: %w", thriftPath, err)
	}
	if len(ast.Services) == 0 {
		return nil, fmt.Errorf("no service found in %s", thriftPath)
	}

	svc := ast.Services[0]
	methods := make([]ServiceMethod, 0, len(svc.Functions))
	for _, fn := range svc.Functions {
		m := ServiceMethod{
			Name:          fn.Name,
			RequestType:   "any",
			ReturnType:    "any",
			RequestTypeTS: "unknown",
			ReturnTypeTS:  "unknown",
		}
		if len(fn.Arguments) == 1 && fn.Arguments[0].Type != nil {
			m.RequestType = thriftTypeToGo(fn.Arguments[0].Type)
			m.RequestTypeTS = tsTypeName(fn.Arguments[0].Type)
		}
		if !fn.Void && fn.FunctionType != nil {
			m.ReturnType = thriftTypeToGo(fn.FunctionType)
			m.ReturnTypeTS = tsTypeName(fn.FunctionType)
		}
		if fn.Void {
			m.ReturnType = "struct{}"
			m.ReturnTypeTS = "void"
		}
		methods = append(methods, m)
	}

	// Extract struct definitions
	structs := make([]StructDef, 0, len(ast.Structs))
	for _, s := range ast.Structs {
		sd := StructDef{Name: s.Name}
		for _, f := range s.Fields {
			sd.Fields = append(sd.Fields, StructField{
				Name:    toPascalCase(f.Name),
				GoType:  thriftTypeToGo(f.Type),
				JSONTag: buildJSONTag(f.Name, f.Requiredness == parser.FieldType_Optional),
			})
		}
		structs = append(structs, sd)
	}

	pkgName, ok := ast.GetNamespace("go")
	if !ok || strings.TrimSpace(pkgName) == "" {
		pkgName = strings.TrimSuffix(filepath.Base(thriftPath), ".thrift")
	}

	return &ThriftServiceInfo{
		PackageName: normalizePackageName(pkgName),
		ServiceName: svc.Name,
		Methods:     methods,
		Structs:     structs,
	}, nil
}

func writeGeneratedClientFiles(goDir, tsDir string, info *ThriftServiceInfo) error {
	goBody, err := gtmpl.RenderFileTemplate("api_client_go.tmpl", info)
	if err != nil {
		return err
	}
	tsBody, err := gtmpl.RenderFileTemplate("api_client_ts.tmpl", info)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(goDir, "client.go"), goBody, 0644); err != nil {
		return fmt.Errorf("write client.go: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tsDir, "client.ts"), tsBody, 0644); err != nil {
		return fmt.Errorf("write client.ts: %w", err)
	}
	return nil
}

func normalizePackageName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return replacer.Replace(s)
}

// toPascalCase converts snake_case to PascalCase, handling common abbreviations.
func toPascalCase(s string) string {
	abbreviations := map[string]string{
		"id": "ID", "url": "URL", "http": "HTTP", "https": "HTTPS",
		"api": "API", "ip": "IP", "uri": "URI", "uid": "UID",
		"uuid": "UUID", "sql": "SQL", "ssh": "SSH", "tcp": "TCP",
		"udp": "UDP", "cpu": "CPU", "gpu": "GPU",
	}

	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if abbr, ok := abbreviations[lower]; ok {
			b.WriteString(abbr)
		} else {
			runes := []rune(lower)
			runes[0] = unicode.ToUpper(runes[0])
			b.WriteString(string(runes))
		}
	}
	return b.String()
}

// thriftTypeToGo recursively maps a Thrift type to a Go type string.
func thriftTypeToGo(t *parser.Type) string {
	if t == nil {
		return "any"
	}
	name := strings.ToLower(strings.TrimSpace(t.Name))
	switch name {
	case "bool":
		return "bool"
	case "byte", "i8":
		return "int8"
	case "i16":
		return "int16"
	case "i32":
		return "int32"
	case "i64":
		return "int64"
	case "double":
		return "float64"
	case "string":
		return "string"
	case "binary":
		return "[]byte"
	case "list", "set":
		elem := "any"
		if t.ValueType != nil {
			elem = thriftTypeToGo(t.ValueType)
		}
		return "[]" + elem
	case "map":
		key := "string"
		val := "any"
		if t.KeyType != nil {
			key = thriftTypeToGo(t.KeyType)
		}
		if t.ValueType != nil {
			val = thriftTypeToGo(t.ValueType)
		}
		return "map[" + key + "]" + val
	default:
		// Struct reference — use the type name as-is (PascalCase from Thrift IDL)
		return strings.TrimSpace(t.Name)
	}
}

// buildJSONTag returns a full Go struct tag for JSON serialization.
func buildJSONTag(fieldName string, optional bool) string {
	tag := fieldName
	if optional {
		tag += ",omitempty"
	}
	return "`json:\"" + tag + "\"`"
}

func tsTypeName(t *parser.Type) string {
	if t == nil {
		return "unknown"
	}
	name := strings.ToLower(strings.TrimSpace(t.Name))
	switch name {
	case "bool":
		return "boolean"
	case "byte", "i8", "i16", "i32", "i64", "double":
		return "number"
	case "string", "binary":
		return "string"
	case "void":
		return "void"
	case "list", "set":
		return "unknown[]"
	case "map":
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}
