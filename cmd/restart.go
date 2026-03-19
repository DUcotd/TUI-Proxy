package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Mihomo 服务",
	RunE:  runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 正在重启 Mihomo...")

	// Try systemd first
	if mihomo.HasSystemd() {
		if err := mihomo.RestartService(core.DefaultServiceName); err == nil {
			fmt.Println("✅ Mihomo 已重启")
			return nil
		} else {
			fmt.Printf("⚠️  systemd 重启失败: %v\n正在回退到进程模式...\n", err)
		}
	}

	// Fallback: kill existing processes and start new one
	mihomo.KillExistingMihomo()

	proc := mihomo.NewProcess(core.DefaultConfigDir)
	if err := proc.Start(); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已重启")
	return nil
}
