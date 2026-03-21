// Package system provides HTTP download helpers.
package system

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FetchJSON fetches a JSON document and decodes it into dest.
func FetchJSON(url string, timeout time.Duration, dest any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return FetchJSONWithDoer(NewHTTPClient(timeout, false), req, dest)
}

// FetchJSONWithDoer fetches a JSON document with the provided HTTP client.
func FetchJSONWithDoer(doer HTTPDoer, req *http.Request, dest any) error {
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from url to destPath.
func DownloadFile(url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return DownloadFileWithDoer(NewHTTPClient(5*time.Minute, false), req, destPath)
}

// DownloadFileWithDoer downloads a file using the provided HTTP client.
func DownloadFileWithDoer(doer HTTPDoer, req *http.Request, destPath string) error {
	return DownloadFileWithOptions(doer, req, destPath, DownloadOptions{})
}

// DownloadOptions controls download behavior.
type DownloadOptions struct {
	ExpectedSHA256 string
	Atomic         bool
}

// DownloadBytes fetches a URL and returns its body.
func DownloadBytes(url string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DownloadBytesWithDoer(NewHTTPClient(timeout, false), req)
}

// DownloadBytesWithDoer fetches a URL using the provided HTTP client.
func DownloadBytesWithDoer(doer HTTPDoer, req *http.Request) ([]byte, error) {
	resp, err := doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// DownloadFileWithOptions downloads a file using the provided HTTP client and options.
func DownloadFileWithOptions(doer HTTPDoer, req *http.Request, destPath string, opts DownloadOptions) error {
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	path := destPath
	cleanupPath := ""
	if opts.Atomic {
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		tmpFile, err := os.CreateTemp(filepath.Dir(destPath), filepath.Base(destPath)+".tmp-*")
		if err != nil {
			return err
		}
		path = tmpFile.Name()
		cleanupPath = path
		_ = tmpFile.Close()
		defer os.Remove(cleanupPath)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.Writer(out)
	if opts.ExpectedSHA256 != "" {
		writer = io.MultiWriter(out, hasher)
	}

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	if opts.ExpectedSHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, opts.ExpectedSHA256) {
			return fmt.Errorf("SHA256 校验失败: got %s want %s", got, opts.ExpectedSHA256)
		}
	}

	if opts.Atomic {
		if err := os.Rename(path, destPath); err != nil {
			return err
		}
	}
	return nil
}
