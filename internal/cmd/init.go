package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/template"
	"github.com/spf13/cobra"
)

var validProjectName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <project-name>",
		Short: i18n.T("init_short"),
		Args:  cobra.ExactArgs(1),
		RunE:  runInit,
	}

	cmd.Flags().String("scaffold", "default", i18n.T("init_flag_scaffold"))

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	scaffoldName, _ := cmd.Flags().GetString("scaffold")

	if !validProjectName.MatchString(projectName) {
		return fmt.Errorf("%s", i18n.Tf("init_invalid_name", projectName))
	}

	projectDir, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("%s", i18n.Tf("init_dir_exists", projectName))
	}

	cfg := config.Default()

	fmt.Println(i18n.Tf("init_creating", projectName))

	// Step 1: Scaffold Go backend
	fmt.Println(i18n.T("init_backend_skeleton"))
	if err := template.Scaffold(projectDir, template.ScaffoldData{
		ProjectName: projectName,
		UIRegistry:  cfg.UIRegistry,
		APIRegistry: cfg.APIRegistry,
	}); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	// Step 2: Initialize frontend from scaffold asset
	scaffoldKey := "scaffold/" + scaffoldName
	fmt.Println(i18n.Tf("init_frontend_from", scaffoldKey))
	if err := initFrontend(projectDir, projectName, cfg, scaffoldKey); err != nil {
		return fmt.Errorf("init frontend: %w", err)
	}

	// Step 3: Generate API artifacts from default hello.thrift
	fmt.Println(i18n.T("init_api_artifacts"))
	thriftPath := filepath.Join(projectDir, "api", projectName, "hello", "v1", "hello.thrift")
	if err := asset.GenerateThriftArtifacts(projectDir, thriftPath); err != nil {
		return fmt.Errorf("generate hello API: %w", err)
	}

	fmt.Println(i18n.Tf("init_success", projectName))
	fmt.Println(i18n.T("init_next_steps"))
	fmt.Println(i18n.Tf("init_next_cd", projectName))
	fmt.Println(i18n.T("init_next_dev"))

	return nil
}
