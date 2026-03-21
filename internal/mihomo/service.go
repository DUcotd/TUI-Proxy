// Package mihomo provides systemd service management for Mihomo.
package mihomo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

// DefaultServiceName is the managed systemd service name.
const DefaultServiceName = core.DefaultServiceName

const serviceTemplate = `[Unit]
Description=Mihomo Proxy Service (managed by clashctl)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{systemdQuote .Binary}} -d {{systemdQuote .ConfigDir}}
Restart=always
RestartSec=3
LimitNOFILE=65535
{{if .User}}User={{.User}}
{{end}}{{if .Group}}Group={{.Group}}
{{end}}[Install]
WantedBy=multi-user.target
`

// ServiceConfig holds parameters for generating the systemd service file.
type ServiceConfig struct {
	Binary      string
	ConfigDir   string
	ServiceName string
	User        string
	Group       string
}

// GenerateServiceFile writes a systemd service file to the appropriate location.
func GenerateServiceFile(cfg ServiceConfig) error {
	data, err := renderServiceFile(cfg)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.ServiceName)
	return writeFileAtomic(path, data, 0644)
}

func renderServiceFile(cfg ServiceConfig) ([]byte, error) {
	tmpl, err := template.New("service").Funcs(template.FuncMap{
		"systemdQuote": systemdQuote,
	}).Parse(serviceTemplate)
	if err != nil {
		return nil, fmt.Errorf("解析服务模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("写入服务文件失败: %w", err)
	}

	return buf.Bytes(), nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建服务目录失败: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建服务文件 %s 失败: %w", path, err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("写入服务文件失败: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("同步服务文件失败: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭服务文件失败: %w", err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("设置服务文件权限失败: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("落盘服务文件失败: %w", err)
	}

	dirHandle, err := os.Open(filepath.Dir(path))
	if err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	return nil
}

func systemdQuote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `	`, ` `)
	return `"` + replacer.Replace(value) + `"`
}

// ReloadSystemd runs systemctl daemon-reload.
func ReloadSystemd() error {
	return system.RunCommandSilent("systemctl", "daemon-reload")
}

// EnableService enables a systemd service.
func EnableService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "enable", serviceName)
	return err
}

// DisableService disables a systemd service.
func DisableService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "disable", serviceName)
	return err
}

// StartService starts a systemd service.
func StartService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "start", serviceName)
	return err
}

// StopService stops a systemd service.
func StopService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "stop", serviceName)
	return err
}

// RestartService restarts a systemd service.
func RestartService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "restart", serviceName)
	return err
}

// HasSystemd checks if systemd is available on this system.
func HasSystemd() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

// ServiceStatus checks if a systemd service is active.
func ServiceStatus(serviceName string) (bool, error) {
	output, err := system.RunCommand("systemctl", "is-active", serviceName)
	if err != nil {
		// "inactive" returns exit code 3, which is not an error for us
		if output == "inactive" || output == "unknown" {
			return false, nil
		}
		return false, err
	}
	return output == "active", nil
}

// SetupSystemd performs the full systemd setup: generate, reload, sync boot policy, and optionally start.
func SetupSystemd(cfg ServiceConfig, enableOnBoot bool, startNow bool) error {
	// Generate service file
	if err := GenerateServiceFile(cfg); err != nil {
		return err
	}

	// Reload systemd
	if err := ReloadSystemd(); err != nil {
		return fmt.Errorf("systemctl daemon-reload 失败: %w", err)
	}

	if enableOnBoot {
		if err := EnableService(cfg.ServiceName); err != nil {
			return fmt.Errorf("systemctl enable 失败: %w", err)
		}
	} else {
		if err := DisableService(cfg.ServiceName); err != nil {
			return fmt.Errorf("systemctl disable 失败: %w", err)
		}
	}

	// Start if requested
	if startNow {
		if err := StartService(cfg.ServiceName); err != nil {
			return fmt.Errorf("systemctl start 失败: %w", err)
		}
	}

	return nil
}
