package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:    "config",
	Short:  "管理配置",
	Hidden: true,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前 Mihomo 配置",
	RunE:  legacyRunner("clashctl config show", "clashctl advanced config show", runConfigShow),
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE:  legacyRunner("clashctl config path", "clashctl advanced config path", runConfigPath),
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	configPath := mihomoConfigPath(cfg)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("无法读取配置文件 %s: %w", configPath, err)
	}
	fmt.Println(string(data))
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	fmt.Printf("配置目录: %s\n", cfg.ConfigDir)
	fmt.Printf("配置文件: %s\n", mihomoConfigPath(cfg))
	fmt.Printf("Provider: %s\n", mihomoProviderPath(cfg))
	return nil
}
