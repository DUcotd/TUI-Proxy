package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
)

var (
	configShowJSON bool
	configPathJSON bool
)

type configShowReport struct {
	ConfigPath string `json:"config_path"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}

type configPathReport struct {
	ConfigDir    string `json:"config_dir"`
	ConfigPath   string `json:"config_path"`
	ProviderPath string `json:"provider_path"`
	Error        string `json:"error,omitempty"`
}

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
	bindConfigFlags(configShowCmd, configPathCmd)
	bindConfigFlags(advancedConfigShowCmd, advancedConfigPathCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}

func bindConfigFlags(showCmd, pathCmd *cobra.Command) {
	showCmd.Flags().BoolVar(&configShowJSON, "json", false, "以 JSON 输出当前配置")
	pathCmd.Flags().BoolVar(&configPathJSON, "json", false, "以 JSON 输出配置路径")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	configPath := mihomoConfigPath(cfg)
	report := &configShowReport{ConfigPath: configPath}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return finishConfigShowReport(report, fmt.Errorf("无法读取配置文件 %s: %w", configPath, err))
	}
	report.Content = string(data)
	if configShowJSON {
		return finishConfigShowReport(report, nil)
	}
	fmt.Println(string(data))
	return finishConfigShowReport(report, nil)
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	report := buildConfigPathReport(cfg)
	if configPathJSON {
		return finishConfigPathReport(report, nil)
	}

	fmt.Printf("配置目录: %s\n", cfg.ConfigDir)
	fmt.Printf("配置文件: %s\n", mihomoConfigPath(cfg))
	fmt.Printf("Provider: %s\n", mihomoProviderPath(cfg))
	return finishConfigPathReport(report, nil)
}

func buildConfigPathReport(cfg *core.AppConfig) *configPathReport {
	return &configPathReport{
		ConfigDir:    cfg.ConfigDir,
		ConfigPath:   mihomoConfigPath(cfg),
		ProviderPath: mihomoProviderPath(cfg),
	}
}

func finishConfigShowReport(report *configShowReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if configShowJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func finishConfigPathReport(report *configPathReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if configPathJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}
