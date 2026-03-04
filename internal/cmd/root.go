package cmd

import (
	"fmt"
	"os"

	"github.com/castle-x/gve/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gve",
	Short: "GVE - Go + Vite + Embed 全栈项目脚手架",
	Long:  "GVE 是一个用于管理 Go + Vite + Embed 全栈项目的 CLI 工具，\n提供项目初始化、UI/API 资产管理、开发服务器和构建等功能。",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Full())
	},
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "管理 UI 资产",
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "管理 API 契约",
}

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "管理资产库 registry",
}

func init() {
	uiCmd.AddCommand(newUIAddCmd())
	uiCmd.AddCommand(newUIListCmd())
	uiCmd.AddCommand(newUISyncCmd())
	uiCmd.AddCommand(newUIDiffCmd())

	apiCmd.AddCommand(newAPIAddCmd())
	apiCmd.AddCommand(newAPISyncCmd())
	apiCmd.AddCommand(newAPINewCmd())
	apiCmd.AddCommand(newAPIGenerateCmd())

	registryCmd.AddCommand(newRegistryBuildCmd())

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
	rootCmd.AddCommand(registryCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
