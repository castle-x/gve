package asset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gtmpl "github.com/castle-x/gve/internal/template"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/sdk"
)

var invokeThriftgo = sdk.InvokeThriftgo
var parseThriftFile = parser.ParseFile

type ServiceMethod struct {
	Name          string
	RequestType   string
	ReturnType    string
	RequestTypeTS string
	ReturnTypeTS  string
}

type ThriftServiceInfo struct {
	PackageName string
	ServiceName string
	Methods     []ServiceMethod
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

	if err := invokeThriftgo(nil, "thriftgo", "-g", "go", "-o", goDir, absPath); err != nil {
		return fmt.Errorf("invoke thriftgo: %w", err)
	}
	baseName := strings.TrimSuffix(filepath.Base(absPath), ".thrift")
	if err := normalizeGeneratedGoFile(goDir, baseName); err != nil {
		return err
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
			m.RequestType = goTypeName(fn.Arguments[0].Type)
			m.RequestTypeTS = tsTypeName(fn.Arguments[0].Type)
		}
		if !fn.Void && fn.FunctionType != nil {
			m.ReturnType = goTypeName(fn.FunctionType)
			m.ReturnTypeTS = tsTypeName(fn.FunctionType)
		}
		if fn.Void {
			m.ReturnType = "struct{}"
			m.ReturnTypeTS = "void"
		}
		methods = append(methods, m)
	}

	pkgName, ok := ast.GetNamespace("go")
	if !ok || strings.TrimSpace(pkgName) == "" {
		pkgName = strings.TrimSuffix(filepath.Base(thriftPath), ".thrift")
	}

	return &ThriftServiceInfo{
		PackageName: normalizePackageName(pkgName),
		ServiceName: svc.Name,
		Methods:     methods,
	}, nil
}

func normalizeGeneratedGoFile(dir, baseName string) error {
	want := filepath.Join(dir, baseName+".go")

	// Always scan for thriftgo-generated files in namespace subdirectories,
	// even if the target already exists. thriftgo recreates the subdirectory
	// on every invocation, so we must clean it up to avoid stale duplicates.
	var found string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != baseName+".go" {
			return nil
		}
		if path == want {
			return nil
		}
		found = path
		return fs.SkipAll
	})
	if err != nil {
		return fmt.Errorf("scan generated go file: %w", err)
	}
	if found == "" {
		if _, err := os.Stat(want); err == nil {
			return nil
		}
		return fmt.Errorf("generated go file not found for %s", baseName)
	}

	_ = os.Remove(want)
	if err := os.Rename(found, want); err != nil {
		return fmt.Errorf("move generated go file: %w", err)
	}
	_ = os.Remove(filepath.Dir(found))
	return nil
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

func goTypeName(t *parser.Type) string {
	if t == nil {
		return "any"
	}
	return strings.TrimSpace(t.Name)
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
