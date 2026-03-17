package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/template"
	"github.com/spf13/cobra"
)

var validProjectName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <project-name>",
		Short: "初始化新项目",
		Args:  cobra.ExactArgs(1),
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	if !validProjectName.MatchString(projectName) {
		return fmt.Errorf("invalid project name %q: must start with a letter and contain only letters, digits, hyphens, underscores", projectName)
	}

	projectDir, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory %q already exists", projectName)
	}

	cfg := config.Default()

	fmt.Printf("Creating project %s...\n", projectName)

	// Step 1: Scaffold Go backend
	fmt.Println("  Generating Go backend skeleton...")
	if err := template.Scaffold(projectDir, template.ScaffoldData{
		ProjectName: projectName,
		UIRegistry:  cfg.UIRegistry,
		APIRegistry: cfg.APIRegistry,
	}); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	// Step 2: Initialize frontend from scaffold asset
	fmt.Println("  Initializing frontend from scaffold asset...")
	if err := initFrontend(projectDir, cfg); err != nil {
		return fmt.Errorf("init frontend: %w", err)
	}

	fmt.Printf("\n✓ Project %s created successfully\n", projectName)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  gve dev")

	return nil
}
