package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	Long:  `自动下载并安装最新版本的 Mihomo 内核到 /usr/local/bin/mihomo。`,
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Check root
	if err := system.RequireRoot(); err != nil {
		return err
	}

	// Check if already installed
	if binary, err := mihomo.FindBinary(); err == nil {
		version, _ := mihomo.GetBinaryVersion()
		fmt.Printf("Mihomo 已安装: %s", binary)
		if version != "" {
			fmt.Printf(" (%s)", version)
		}
		fmt.Println()

		// Ask to reinstall
		fmt.Println("如需重新安装，请先卸载当前版本")
		return nil
	}

	// Download and install
	if _, err := mihomo.InstallMihomo(); err != nil {
		return fmt.Errorf("安装失败: %w", err)
	}

	fmt.Println("")
	fmt.Println("💡 运行 'sudo clashctl init' 开始配置")

	return nil
}
