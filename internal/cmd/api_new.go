package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/castle-x/gve/internal/template"
	"github.com/spf13/cobra"
)

type apiSkeletonData struct {
	Project      string
	Resource     string
	ResourceType string
	NamespaceGo  string
	NamespaceJS  string
}

func newAPINewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <project>/<resource> [version]",
		Short: "在项目内创建 API thrift 骨架",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runAPINew,
	}
}

func runAPINew(cmd *cobra.Command, args []string) error {
	target := strings.TrimSpace(args[0])
	project, resource, err := parseAPIResourceTarget(target)
	if err != nil {
		return err
	}

	version := "v1"
	if len(args) > 1 {
		version = strings.TrimSpace(args[1])
		if version == "" {
			return fmt.Errorf("version cannot be empty")
		}
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	destDir := filepath.Join(projectDir, "api", project, resource, version)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create api directory: %w", err)
	}

	thriftPath := filepath.Join(destDir, resource+".thrift")
	if _, err := os.Stat(thriftPath); err == nil {
		return fmt.Errorf("thrift file already exists: %s", thriftPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check thrift file: %w", err)
	}

	data := apiSkeletonData{
		Project:      project,
		Resource:     resource,
		ResourceType: title(resource),
		NamespaceGo:  normalizeNamespace(resource),
		NamespaceJS:  normalizeNamespace(resource),
	}
	body, err := template.RenderFileTemplate("api_skeleton.thrift.tmpl", data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(thriftPath, body, 0644); err != nil {
		return fmt.Errorf("write thrift skeleton: %w", err)
	}

	fmt.Printf("✓ Created %s\n", thriftPath)
	fmt.Println("Edit the thrift file, then run: gve api generate")
	return nil
}

func parseAPIResourceTarget(target string) (project, resource string, err error) {
	parts := strings.Split(target, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid target %q — expected format: project/resource", target)
	}
	return parts[0], parts[1], nil
}

func normalizeNamespace(s string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(s)))
}

func title(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
