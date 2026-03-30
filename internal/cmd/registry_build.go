package cmd

import (
	"fmt"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/spf13/cobra"
)

func newRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "registry",
		Short: i18n.T("registry_short"),
	}
}

func newRegistryBuildCmd() *cobra.Command {
	var rootDir string
	var category string

	cmd := &cobra.Command{
		Use:   "build",
		Short: i18n.T("registry_build_short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryBuild(rootDir, category)
		},
	}

	cmd.Flags().StringVar(&rootDir, "root", ".", "root directory of the asset registry")
	cmd.Flags().StringVar(&category, "category", "", "comma-separated categories to build (e.g. ui,components)")

	return cmd
}

var validCategories = map[string]bool{
	"scaffold":   true,
	"ui":         true,
	"components": true,
	"hooks":      true,
}

func runRegistryBuild(rootDir, category string) error {
	var reg asset.Registry
	var warnings []string
	var err error

	if category != "" {
		categories := strings.Split(category, ",")
		for i := range categories {
			categories[i] = strings.TrimSpace(categories[i])
		}
		// Validate categories
		for _, c := range categories {
			if !validCategories[c] {
				return fmt.Errorf("%s", i18n.Tf("registry_build_invalid_category", c))
			}
		}

		for _, c := range categories {
			fmt.Println(i18n.Tf("registry_build_scanning", c))
		}

		reg, warnings, err = asset.BuildRegistryV2Filtered(rootDir, categories)
	} else {
		fmt.Println(i18n.Tf("registry_build_scanning", "all categories"))
		reg, warnings, err = asset.BuildRegistryV2(rootDir)
	}
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Println(i18n.Tf("registry_build_warn", w))
	}

	outPath := rootDir + "/registry.json"
	if err := asset.WriteRegistryV2(reg, outPath); err != nil {
		return err
	}

	// Build summary
	counts := make(map[string]int)
	for key := range reg {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			counts[parts[0]]++
		}
	}
	var summary []string
	for cat, n := range counts {
		summary = append(summary, fmt.Sprintf("%s: %d", cat, n))
	}
	fmt.Println(i18n.Tf("registry_build_ok", strings.Join(summary, ", ")))
	return nil
}
