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


var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "启动开发服务器（Air + Vite）",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("dev: not yet implemented")
		return nil
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建单二进制文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("build: not yet implemented")
		return nil
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示资产状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("status: not yet implemented")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "检查开发环境",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("doctor: not yet implemented")
		return nil
	},
}

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "管理资产库 registry",
}

func init() {
	uiCmd.AddCommand(&cobra.Command{
		Use: "add <asset>[@version]", Short: "安装 UI 资产",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("ui add: not yet implemented")
			return nil
		},
	})
	uiCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "列出已安装 UI 资产",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("ui list: not yet implemented")
			return nil
		},
	})
	uiCmd.AddCommand(&cobra.Command{
		Use: "sync [asset]", Short: "同步 UI 资产",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("ui sync: not yet implemented")
			return nil
		},
	})
	uiCmd.AddCommand(&cobra.Command{
		Use: "diff <asset>", Short: "查看资产差异",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("ui diff: not yet implemented")
			return nil
		},
	})

	apiCmd.AddCommand(&cobra.Command{
		Use: "add <project>/<resource>[@version]", Short: "安装 API 契约",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("api add: not yet implemented")
			return nil
		},
	})
	apiCmd.AddCommand(&cobra.Command{
		Use: "sync", Short: "同步 API 契约",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("api sync: not yet implemented")
			return nil
		},
	})

	registryCmd.AddCommand(&cobra.Command{
		Use: "build", Short: "构建 registry.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("registry build: not yet implemented")
			return nil
		},
	})

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(registryCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
