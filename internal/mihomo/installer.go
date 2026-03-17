// Package mihomo provides automatic Mihomo installation.
package mihomo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const (
	// InstallPath is the default location where clashctl installs mihomo.
	InstallPath = "/usr/local/bin/mihomo"
	// MihomoGitHubOwner is the GitHub repo owner for Mihomo releases.
	MihomoGitHubOwner = "MetaCubeX"
	// MihomoGitHubRepo is the GitHub repo name for Mihomo releases.
	MihomoGitHubRepo = "mihomo"
)

// GitHubRelease represents a GitHub release (minimal fields).
type MihomoRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// EnsureMihomo checks if mihomo is available, and if not, downloads and installs it.
// Returns the path to the binary.
func EnsureMihomo() (string, error) {
	// First check if already available
	if path, err := FindBinary(); err == nil {
		return path, nil
	}

	// Not found, need to install
	return InstallMihomo()
}

// InstallMihomo downloads the latest mihomo binary to InstallPath.
func InstallMihomo() (string, error) {
	fmt.Println("📦 Mihomo 未安装，正在自动下载...")

	release, err := fetchLatestMihomoRelease()
	if err != nil {
		return "", fmt.Errorf("获取 Mihomo 版本信息失败: %w", err)
	}

	fmt.Printf("   最新版本: %s\n", release.TagName)

	// Find matching binary
	downloadURL := findMihomoAsset(release)
	if downloadURL == "" {
		return "", fmt.Errorf("未找到适用于 %s/%s 的 Mihomo 二进制文件", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("   下载中: %s\n", downloadURL)

	if err := downloadBinary(downloadURL, InstallPath); err != nil {
		return "", fmt.Errorf("下载 Mihomo 失败: %w", err)
	}

	if err := os.Chmod(InstallPath, 0755); err != nil {
		return "", fmt.Errorf("设置执行权限失败: %w", err)
	}

	fmt.Printf("✅ Mihomo 已安装到: %s\n", InstallPath)

	// Verify
	version, _ := GetBinaryVersion()
	if version != "" {
		fmt.Printf("   版本: %s\n", version)
	}

	return InstallPath, nil
}

// fetchLatestMihomoRelease gets the latest release info from GitHub API.
func fetchLatestMihomoRelease() (*MihomoRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		MihomoGitHubOwner, MihomoGitHubRepo)

	client := &http.Client{Timeout: 15 * 1e9}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 %d", resp.StatusCode)
	}

	var release MihomoRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

// findMihomoAsset finds the correct binary asset for the current platform.
func findMihomoAsset(release *MihomoRelease) string {
	// Mihomo assets typically named like: mihomo-linux-amd64-v1.18.0.gz
	// or mihomo-linux-amd64 (no compression)
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Prefer uncompressed binary first
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) &&
			!strings.HasSuffix(name, ".gz") && !strings.HasSuffix(name, ".zip") {
			return asset.BrowserDownloadURL
		}
	}

	// Then try .gz
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) &&
			strings.HasSuffix(name, ".gz") {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

// downloadBinary downloads a file from url to destPath.
func downloadBinary(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
