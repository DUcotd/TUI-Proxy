// Package config handles reading and writing clashctl configuration files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// BackupFile creates a timestamped backup of an existing file.
// Returns the backup path or an error.
func BackupFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // nothing to back up
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.bak.%s", path, timestamp)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 %s 备份失败: %w", path, err)
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("写入备份到 %s 失败: %w", backupPath, err)
	}

	return backupPath, nil
}

// WriteConfig writes data to a path, creating parent directories if needed.
func WriteConfig(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("写入配置到 %s 失败: %w", path, err)
	}

	return nil
}

// ValidateYAML reads back a YAML file and checks it can be parsed.
func ValidateYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", path, err)
	}

	var dummy any
	if err := yaml.Unmarshal(data, &dummy); err != nil {
		return fmt.Errorf("YAML 解析错误 %s: %w", path, err)
	}

	return nil
}

// Loader provides configuration loading from files.
type Loader struct {
	Path string
}

// NewLoader creates a Loader for the given config file path.
func NewLoader(path string) *Loader {
	return &Loader{Path: path}
}

// Load reads and unmarshals the YAML config file into dest.
func (l *Loader) Load(dest any) error {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return fmt.Errorf("读取配置文件 %s 失败: %w", l.Path, err)
	}
	if err := yaml.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析 YAML 文件 %s 失败: %w", l.Path, err)
	}
	return nil
}
