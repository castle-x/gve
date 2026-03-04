package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/spf13/cobra"
)

var generateThriftArtifacts = asset.GenerateThriftArtifacts

func newAPIGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "生成 API 代码与客户端",
		RunE:    runAPIGenerate,
	}
	return cmd
}

func runAPIGenerate(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	thriftFiles, err := collectCanonicalThriftFiles(projectDir)
	if err != nil {
		return err
	}
	if len(thriftFiles) == 0 {
		fmt.Println("No canonical thrift files found under api/.")
		return nil
	}

	for _, thriftPath := range thriftFiles {
		fmt.Printf("Generating from %s...\n", thriftPath)
		if err := generateThriftArtifacts(projectDir, thriftPath); err != nil {
			return fmt.Errorf("generate %s: %w", thriftPath, err)
		}
	}

	fmt.Printf("✓ Generated API artifacts for %d thrift file(s)\n", len(thriftFiles))
	return nil
}

func collectCanonicalThriftFiles(projectDir string) ([]string, error) {
	apiDir := filepath.Join(projectDir, "api")
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return nil, nil
	}

	var thriftFiles []string
	err := filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".thrift" {
			return nil
		}

		rel, err := filepath.Rel(apiDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 4 {
			return nil
		}

		resource := parts[1]
		version := parts[2]
		file := parts[3]

		if !strings.HasPrefix(version, "v") {
			return nil
		}

		expectedFile := resource + ".thrift"
		if file != expectedFile {
			return fmt.Errorf("invalid thrift file name %q, expected %q in %s", file, expectedFile, filepath.Dir(path))
		}

		thriftFiles = append(thriftFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(thriftFiles)
	return thriftFiles, nil
}
