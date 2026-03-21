package system

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type NamedDownload struct {
	Name string
	URL  string
}

const MaxChecksumFileBytes = 1 * 1024 * 1024

var sha256Pattern = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)

// FindChecksumAsset looks for a checksum artifact that can verify targetName.
func FindChecksumAsset(assets []NamedDownload, targetName string) (NamedDownload, bool) {
	exactNames := map[string]bool{
		targetName + ".sha256":     true,
		targetName + ".sha256sum":  true,
		targetName + ".sha256.txt": true,
	}
	for _, asset := range assets {
		if exactNames[asset.Name] {
			return asset, true
		}
	}
	for _, asset := range assets {
		lower := strings.ToLower(asset.Name)
		if strings.Contains(lower, "checksum") || strings.Contains(lower, "sha256") {
			return asset, true
		}
	}
	return NamedDownload{}, false
}

// ExtractSHA256 finds targetName's checksum from checksum content.
func ExtractSHA256(data []byte, targetName string) (string, error) {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && sha256Pattern.MatchString(fields[0]) {
			candidate := strings.TrimPrefix(fields[len(fields)-1], "*")
			candidate = strings.Trim(candidate, "()")
			if candidate == targetName {
				return strings.ToLower(fields[0]), nil
			}
		}
		prefix := "sha256 (" + strings.ToLower(targetName) + ") = "
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, prefix) {
			hash := strings.TrimSpace(line[len(prefix):])
			if sha256Pattern.MatchString(hash) {
				return strings.ToLower(hash), nil
			}
		}
		if sha256Pattern.MatchString(line) {
			return strings.ToLower(line), nil
		}
	}
	return "", fmt.Errorf("未找到 %s 的 SHA256 校验值", targetName)
}

// DownloadVerifiedFile downloads a file and verifies it against a release checksum artifact.
func DownloadVerifiedFile(asset NamedDownload, checksumAsset NamedDownload, destPath string) error {
	checksumData, err := DownloadBytesLimit(checksumAsset.URL, 2*time.Minute, MaxChecksumFileBytes)
	if err != nil {
		return fmt.Errorf("下载校验文件失败: %w", err)
	}
	want, err := ExtractSHA256(checksumData, asset.Name)
	if err != nil {
		return err
	}
	req, err := newDownloadRequest(asset.URL)
	if err != nil {
		return err
	}
	return DownloadFileWithOptions(NewHTTPClient(5*time.Minute, false), req, destPath, DownloadOptions{
		ExpectedSHA256: want,
		Atomic:         true,
	})
}

func newDownloadRequest(url string) (*http.Request, error) {
	return http.NewRequest(http.MethodGet, url, nil)
}
