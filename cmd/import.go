package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/subscription"
)

var (
	importFile      string
	importOutput    string
	importMode      string
	importMixedPort int
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	Long: `从本地文件导入原始订阅内容并生成可直接运行的 Mihomo 配置。

支持两类输入：
  - base64 编码的原始订阅
  - 解码后的 vless:// / trojan:// / hysteria2:// 链接列表`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "本地订阅文件路径（必填）")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "config.yaml", "输出文件路径")
	importCmd.Flags().StringVarP(&importMode, "mode", "m", "mixed", "运行模式: tun 或 mixed")
	importCmd.Flags().IntVarP(&importMixedPort, "port", "p", core.DefaultMixedPort, "mixed-port 值")
	importCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(importFile)
	if err != nil {
		return fmt.Errorf("读取订阅文件失败: %w", err)
	}

	parsed, err := subscription.Parse(data)
	if err != nil {
		return fmt.Errorf("解析订阅文件失败: %w", err)
	}

	cfg := core.DefaultAppConfig()
	cfg.Mode = importMode
	cfg.MixedPort = importMixedPort

	mihomoCfg := core.BuildStaticMihomoConfig(cfg, parsed.Proxies, parsed.Names)
	yamlData, err := core.RenderYAML(mihomoCfg)
	if err != nil {
		return fmt.Errorf("YAML 渲染失败: %w", err)
	}

	if err := os.WriteFile(importOutput, yamlData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("✅ 配置已导出到: %s\n", importOutput)
	fmt.Printf("   来源格式: %s\n", parsed.DetectedFormat)
	fmt.Printf("   节点数量: %d\n", len(parsed.Names))
	fmt.Printf("   模式: %s\n", cfg.Mode)
	fmt.Println("   说明: 这是静态配置，不依赖服务器再次拉取订阅 URL")

	return nil
}
