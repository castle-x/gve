package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/castle-x/gve/internal/asset"
	"github.com/spf13/cobra"
)

func newRegistryBuildCmd() *cobra.Command {
	var assetsDir string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "扫描 assets/ 目录，生成 registry.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryBuild(assetsDir)
		},
	}

	cmd.Flags().StringVar(&assetsDir, "dir", "assets", "资产目录路径")
	return cmd
}

func runRegistryBuild(assetsDir string) error {
	absDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	fmt.Printf("Scanning %s ...\n", absDir)

	reg, err := asset.BuildRegistry(absDir)
	if err != nil {
		return fmt.Errorf("build registry: %w", err)
	}

	outPath := "registry.json"
	if err := asset.WriteRegistry(reg, outPath); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	fmt.Printf("\nGenerated %s:\n", outPath)
	for name, info := range reg {
		fmt.Printf("  %-20s latest: %s  (%d versions)\n", name, info.Latest, len(info.Versions))
	}

	return nil
}
