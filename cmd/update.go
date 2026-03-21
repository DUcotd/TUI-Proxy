package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

const (
	githubOwner = "DUcotd"
	githubRepo  = "clashctl"
)

// currentVer references the canonical version from core.
var currentVer = core.AppVersion

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "检查并更新 clashctl",
	Long:  `检查 GitHub Releases 获取最新版本，如有更新则自动下载替换。`,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 当前版本: %s\n\n", currentVer)
	fmt.Println("正在检查更新...")

	// Fetch latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}

	latestTag := release.TagName
	fmt.Printf("   最新版本: %s\n", latestTag)

	if latestTag == currentVer {
		fmt.Println("\n✅ 已是最新版本！")
		return nil
	}

	fmt.Printf("\n🆕 发现新版本: %s → %s\n", currentVer, latestTag)

	// Find the right binary for current platform
	binaryName := fmt.Sprintf("clashctl-%s-%s", runtime.GOOS, runtime.GOARCH)
	downloadURL := ""
	checksumAsset := system.NamedDownload{}

	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("未找到适用于 %s/%s 的二进制文件", runtime.GOOS, runtime.GOARCH)
	}
	assets := make([]system.NamedDownload, 0, len(release.Assets))
	for _, asset := range release.Assets {
		assets = append(assets, system.NamedDownload{Name: asset.Name, URL: asset.BrowserDownloadURL})
	}
	var ok bool
	checksumAsset, ok = system.FindChecksumAsset(assets, binaryName)
	if !ok {
		return fmt.Errorf("发布缺少 %s 的校验文件", binaryName)
	}

	fmt.Printf("   下载地址: %s\n", downloadURL)

	// Get current binary path
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前程序路径: %w", err)
	}

	// Check write permission
	if !system.IsRoot() {
		fmt.Println("\n⚠️  更新需要 root 权限")
		fmt.Println("请使用 sudo clashctl update")
		return fmt.Errorf("权限不足")
	}

	fmt.Println("\n正在下载更新...")

	// Download new binary to temp file
	tmpPath := selfPath + ".tmp"
	asset := system.NamedDownload{Name: binaryName, URL: downloadURL}
	if err := downloadVerifiedReleaseAsset(asset, checksumAsset, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载失败: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("设置权限失败: %w", err)
	}
	if err := validateDownloadedClashctlBinary(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载的 clashctl 二进制不可用: %w", err)
	}

	// Replace current binary
	// Backup old one first
	backupPath := selfPath + ".bak"
	if err := os.Rename(selfPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("备份旧版本失败: %w", err)
	}

	if err := os.Rename(tmpPath, selfPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, selfPath)
		return fmt.Errorf("替换文件失败: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	fmt.Printf("\n✅ 更新完成！\n")
	fmt.Printf("   %s → %s\n", currentVer, latestTag)
	fmt.Println("\n运行 'clashctl --help' 查看新版本功能")

	return nil
}

func fetchLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	var release GitHubRelease
	if err := system.FetchJSON(url, 10*time.Second, &release); err != nil {
		return nil, fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	return &release, nil
}

func downloadVerifiedReleaseAsset(asset, checksumAsset system.NamedDownload, destPath string) error {
	if err := system.DownloadVerifiedFile(asset, checksumAsset, destPath); err != nil {
		mirrorAsset := asset
		mirrorAsset.URL = mihomo.GetGitHubMirrorURL(asset.URL)
		mirrorChecksum := checksumAsset
		mirrorChecksum.URL = mihomo.GetGitHubMirrorURL(checksumAsset.URL)
		if mirrorAsset.URL != asset.URL {
			if mirrorErr := system.DownloadVerifiedFile(mirrorAsset, mirrorChecksum, destPath); mirrorErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

func validateDownloadedClashctlBinary(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "version")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("执行 version 超时")
	}
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("version 执行失败: %s", msg)
	}
	if !strings.Contains(string(output), "clashctl ") {
		return fmt.Errorf("version 输出异常: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
