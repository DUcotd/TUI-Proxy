package cmd

import "github.com/spf13/cobra"

var advancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "高级命令与脚本化入口",
	Long:  `集中放置安装、导入导出和配置查看等高级/脚本化能力。`,
}

var advancedInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	RunE:  runInstall,
}

var advancedExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出 Mihomo 配置文件",
	RunE:  runExport,
}

var advancedImportCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	RunE:  runImport,
}

var advancedConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "查看当前配置与路径",
}

var advancedConfigShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前 Mihomo 配置",
	RunE:  runConfigShow,
}

var advancedConfigPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE:  runConfigPath,
}

func init() {
	bindExportFlags(advancedExportCmd)
	bindImportFlags(advancedImportCmd)

	advancedConfigCmd.AddCommand(advancedConfigShowCmd)
	advancedConfigCmd.AddCommand(advancedConfigPathCmd)
	advancedCmd.AddCommand(advancedInstallCmd)
	advancedCmd.AddCommand(advancedExportCmd)
	advancedCmd.AddCommand(advancedImportCmd)
	advancedCmd.AddCommand(advancedConfigCmd)
	rootCmd.AddCommand(advancedCmd)
}
