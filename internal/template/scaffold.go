package template

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed files/*
var templateFS embed.FS

type ScaffoldData struct {
	ProjectName string
	UIRegistry  string
	APIRegistry string
}

type fileMapping struct {
	tmplName string
	destPath string
}

func Scaffold(projectDir string, data ScaffoldData) error {
	mappings := []fileMapping{
		{"go.mod.tmpl", "go.mod"},
		{"main.go.tmpl", "cmd/server/main.go"},
		{"Makefile.tmpl", "Makefile"},
		{"gitignore.tmpl", ".gitignore"},
		{"gve.lock.tmpl", "gve.lock"},
		{"hello_handler.go.tmpl", "internal/handler/hello_handler.go"},
		{"hello.thrift.tmpl", "api/" + data.ProjectName + "/hello/v1/hello.thrift"},
	}

	for _, m := range mappings {
		destPath := filepath.Join(projectDir, m.destPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", m.destPath, err)
		}

		content, err := templateFS.ReadFile("files/" + m.tmplName)
		if err != nil {
			return fmt.Errorf("read template %s: %w", m.tmplName, err)
		}

		tmpl, err := template.New(m.tmplName).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", m.tmplName, err)
		}

		f, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", m.destPath, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("execute template %s: %w", m.tmplName, err)
		}
		f.Close()
	}

	// Create placeholder directories with .gitkeep
	keepDirs := []string{
		"internal/service",
		"api",
	}
	for _, d := range keepDirs {
		dir := filepath.Join(projectDir, d)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
		gitkeep := filepath.Join(dir, ".gitkeep")
		if err := os.WriteFile(gitkeep, nil, 0644); err != nil {
			return fmt.Errorf("create %s/.gitkeep: %w", d, err)
		}
	}

	return nil
}

// RenderFileTemplate renders an embedded template from internal/template/files.
func RenderFileTemplate(name string, data interface{}) ([]byte, error) {
	content, err := templateFS.ReadFile("files/" + name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}
