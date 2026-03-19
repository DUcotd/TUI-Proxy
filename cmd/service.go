package cmd

import "github.com/spf13/cobra"

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "管理 Mihomo 服务",
	Long:  `统一管理 Mihomo 的启动、停止、重启和状态查看。`,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 服务",
	RunE:  runStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 服务",
	RunE:  runStop,
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Mihomo 服务",
	RunE:  runRestart,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Mihomo 运行状态",
	RunE:  runStatus,
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}
