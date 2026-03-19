// Package mihomo provides process management for Mihomo.
package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"clashctl/internal/core"
)

const (
	// StartupWait is the time to wait after starting before checking if process is alive.
	StartupWait = 500 * time.Millisecond
	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	ShutdownTimeout = 5 * time.Second
	// KillWait is the time to wait after killing processes.
	KillWait = 1 * time.Second
)

// Process manages a Mihomo child process.
type Process struct {
	ConfigDir string
	cmd       *exec.Cmd
	devNull   *os.File
}

// NewProcess creates a new Process manager.
func NewProcess(configDir string) *Process {
	return &Process{ConfigDir: configDir}
}

// Start launches Mihomo as a background daemon process.
// It redirects stdout/stderr to /dev/null and creates a new process group
// so the process survives when the parent exits.
func (p *Process) Start() error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	p.cmd = exec.Command(binary, "-d", p.ConfigDir)

	// Open /dev/null for redirecting child process output.
	// After Start(), we close the parent's copy — the child inherits its own FD via fork.
	devNull, devErr := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if devErr != nil {
		p.cmd.Stdout = nil
		p.cmd.Stderr = nil
	} else {
		p.cmd.Stdout = devNull
		p.cmd.Stderr = devNull
	}

	// Create new process group to detach from parent.
	// Note: Setsid is blocked in some container environments (CAP_SYS_ADMIN required),
	// so we only use Setpgid which works everywhere.
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Detach stdin
	p.cmd.Stdin = nil

	if err := p.cmd.Start(); err != nil {
		// Close devNull since child didn't start — FD would leak otherwise
		if devErr == nil {
			devNull.Close()
		}
		return fmt.Errorf("启动 Mihomo 失败: %w", err)
	}

	// Close parent's copy of devNull — child has its own FD via fork.
	// This prevents FD leak on repeated start/stop cycles.
	if devErr == nil {
		devNull.Close()
	}

	// Give it a moment to start up
	time.Sleep(StartupWait)

	// Check if it actually started (use process.Signal(0) which still works)
	if p.cmd.Process == nil {
		return fmt.Errorf("Mihomo 进程启动后立即退出")
	}
	if err := p.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("Mihomo 进程启动后立即退出: %w", err)
	}

	return nil
}

// Stop terminates the Mihomo process.
func (p *Process) Stop() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return p.cmd.Process.Kill()
	}

	// Wait up to ShutdownTimeout for graceful exit
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(ShutdownTimeout):
		// Force kill if graceful shutdown timed out
		if err := p.cmd.Process.Kill(); err != nil {
			return err
		}
		// Wait for Wait goroutine to complete after kill
		<-done
		return nil
	}
}

// IsRunning checks if the Mihomo process is still alive.
// Note: After Setsid/detach, this only works if we still have the pid reference.
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// IsMihomoRunning checks if ANY mihomo process is running (system-wide)
// by attempting to reach the default controller API.
func IsMihomoRunning() bool {
	return IsMihomoRunningAt(core.DefaultControllerAddr)
}

// IsMihomoRunningAt checks if mihomo is running at the given controller address.
func IsMihomoRunningAt(controllerAddr string) bool {
	client := NewClient("http://" + controllerAddr)
	return client.CheckConnection() == nil
}

// KillExistingMihomo kills any running mihomo processes to free the port.
// Returns true if processes were killed, false if none were found.
func KillExistingMihomo() bool {
	// Use pkill to find and kill mihomo processes
	cmd := exec.Command("pkill", "-9", "mihomo")
	err := cmd.Run()
	if err != nil {
		// pkill returns non-zero if no processes matched - that's fine
		return false
	}
	// Give processes time to die and release ports
	time.Sleep(KillWait)
	return true
}

// FindBinary locates the mihomo binary in PATH or at the default install location.
func FindBinary() (string, error) {
	// Try "mihomo" first, then "clash-meta", then "clash"
	for _, name := range []string{"mihomo", "clash-meta", "clash"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Fall back to clashctl's default install path
	if _, err := os.Stat(InstallPath); err == nil {
		return InstallPath, nil
	}

	return "", fmt.Errorf("未找到 mihomo 可执行文件。请先安装 Mihomo 并确保其在 PATH 中")
}

// GetBinaryVersion returns the version string of the mihomo binary.
func GetBinaryVersion() (string, error) {
	binary, err := FindBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(binary, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取版本号失败: %w", err)
	}

	// Version is typically the first line
	version := string(output)
	if len(version) > 100 {
		version = version[:100]
	}

	return version, nil
}
