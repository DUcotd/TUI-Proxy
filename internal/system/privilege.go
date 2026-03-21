// Package system provides privilege and permission utilities.
package system

import (
	"fmt"
	"os"
	"path/filepath"
)

// IsRoot checks if the current process is running as root.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// RequireRoot returns an error if not running as root.
func RequireRoot() error {
	if !IsRoot() {
		return fmt.Errorf("此操作需要 root 权限，请使用 sudo 运行")
	}
	return nil
}

// RequireRootForOperation returns an error with a specific operation name.
func RequireRootForOperation(operation string) error {
	if !IsRoot() {
		return fmt.Errorf("操作 %q 需要 root 权限，请使用 sudo 运行", operation)
	}
	return nil
}

// SuggestSudo returns a message suggesting to use sudo.
func SuggestSudo(command string) string {
	return fmt.Sprintf("请使用 sudo 运行: sudo %s", command)
}

// CanWritePath checks if the current user can write to a path.
func CanWritePath(path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s 是目录，无法按文件写入", path)
		}
		f, openErr := os.OpenFile(path, os.O_WRONLY, 0)
		if openErr != nil {
			if os.IsPermission(openErr) {
				return fmt.Errorf("没有写入 %s 的权限", path)
			}
			return openErr
		}
		return f.Close()
	} else if !os.IsNotExist(err) {
		if os.IsPermission(err) {
			return fmt.Errorf("没有访问 %s 的权限", path)
		}
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".perm-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("没有写入 %s 的权限", path)
		}
		return err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(tmpPath)
	return nil
}
