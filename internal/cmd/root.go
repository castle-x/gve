package cmd

import (
	"fmt"
	"os"

	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/version"
	"github.com/spf13/cobra"
)

func Execute() {
	i18n.MustInit()

	rootCmd := &cobra.Command{
		Use:                   "gve",
		Short:                 i18n.T("root_short"),
		Long:                  i18n.T("root_long"),
		CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: i18n.T("version_short"),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Full())
		},
	}

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: i18n.T("ui_short"),
	}

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: i18n.T("api_short"),
	}

	uiCmd.AddCommand(newUIAddCmd())
	uiCmd.AddCommand(newUIListCmd())
	uiCmd.AddCommand(newUIUpdateCmd())
	uiCmd.AddCommand(newUIDiffCmd())
	uiCmd.AddCommand(newUIPushCmd())

	apiCmd.AddCommand(newAPIAddCmd())
	apiCmd.AddCommand(newAPIUpdateCmd())
	apiCmd.AddCommand(newAPINewCmd())
	apiCmd.AddCommand(newAPIGenerateCmd())
	apiCmd.AddCommand(newAPIPushCmd())

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newDoctorCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
