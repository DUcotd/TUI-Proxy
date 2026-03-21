package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var installJSON bool

type installRunReport struct {
	Installed     bool               `json:"installed"`
	AlreadyExists bool               `json:"already_exists"`
	RequiresRoot  bool               `json:"requires_root,omitempty"`
	Binary        *installJSONReport `json:"binary,omitempty"`
	Error         string             `json:"error,omitempty"`
}

var installCmd = &cobra.Command{
	Use:    "install",
	Short:  "安装 Mihomo 内核",
	Long:   `自动下载并安装最新版本的 Mihomo 内核到 /usr/local/bin/mihomo。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl install", "clashctl advanced install", runInstall),
}

func init() {
	bindInstallFlags(installCmd)
	bindInstallFlags(advancedInstallCmd)
	rootCmd.AddCommand(installCmd)
}

func bindInstallFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&installJSON, "json", false, "以 JSON 输出安装结果")
}

func runInstall(cmd *cobra.Command, args []string) error {
	report := &installRunReport{}
	// Check root
	if err := system.RequireRoot(); err != nil {
		report.RequiresRoot = true
		return finishInstallReport(report, err)
	}

	// Check if already installed
	if binary, err := mihomo.FindBinary(); err == nil {
		version, _ := mihomo.GetBinaryVersion()
		report.AlreadyExists = true
		report.Binary = &installJSONReport{Path: binary, Version: version, Installed: false}
		if !installJSON {
			printInstallStatus(os.Stdout, binary, version)
		}
		return finishInstallReport(report, nil)
	}

	// Download and install
	result, err := mihomo.InstallMihomo()
	if err != nil {
		return finishInstallReport(report, fmt.Errorf("安装失败: %w", err))
	}
	report.Installed = true
	report.Binary = &installJSONReport{
		Path:       result.Path,
		Version:    result.Version,
		ReleaseTag: result.ReleaseTag,
		Installed:  result.Installed,
	}
	if !installJSON {
		printInstallResult(os.Stdout, result)
	}

	return finishInstallReport(report, nil)
}

func finishInstallReport(report *installRunReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if installJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}
