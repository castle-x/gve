package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/spf13/cobra"
)

func newRegistryBuildCmd() *cobra.Command {
	var rootDir string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "扫描 scaffold/ui/components/global/ 目录，生成 registry.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryBuild(rootDir)
		},
	}

	cmd.Flags().StringVar(&rootDir, "root", ".", "仓库根目录路径")
	return cmd
}

func runRegistryBuild(rootDir string) error {
	absDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	fmt.Printf("Scanning %s ...\n", absDir)

	reg, warnings, err := asset.BuildRegistryV2(absDir)
	if err != nil {
		return fmt.Errorf("build registry: %w", err)
	}

	for _, w := range warnings {
		fmt.Printf("  %s\n", w)
	}

	outPath := filepath.Join(absDir, "registry.json")
	if err := asset.WriteRegistryV2(reg, outPath); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	// Count by category
	counts := map[string]int{}
	for key := range reg {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			counts[parts[0]]++
		}
	}

	for _, cat := range []string{"scaffold", "ui", "components", "global"} {
		if c := counts[cat]; c > 0 {
			fmt.Printf("  %s/: %d assets\n", cat, c)
		}
	}

	fmt.Printf("\n✓ registry.json updated (%d assets, %d warnings)\n", len(reg), len(warnings))
	return nil
}
