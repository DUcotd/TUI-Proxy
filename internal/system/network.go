// Package system provides network utility functions.
package system

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"clashctl/internal/core"
)

type URLProbeResult struct {
	StatusCode  int
	ContentKind string
	BodyPreview string
}

// CheckURLReachable performs an HTTP GET request to verify a URL is accessible.
// Uses GET instead of HEAD because many subscription servers don't support HEAD.
func CheckURLReachable(rawURL string, timeout time.Duration) error {
	_, err := ProbeURL(rawURL, timeout)
	return err
}

// ProbeURL fetches a URL with a lightweight GET and classifies the response shape.
func ProbeURL(rawURL string, timeout time.Duration) (*URLProbeResult, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("无法构建请求: %w", err)
	}
	// Some providers require a User-Agent to return proper content
	req.Header.Set("User-Agent", "clashctl/"+core.AppVersion)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法访问 %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s 返回 HTTP %d", rawURL, resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &URLProbeResult{
		StatusCode:  resp.StatusCode,
		ContentKind: classifyBody(body),
		BodyPreview: string(body),
	}, nil
}

func classifyBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "empty"
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<html") || strings.Contains(lower, "<body") {
		return "html"
	}
	if strings.Contains(trimmed, "proxies:") || strings.Contains(trimmed, "proxy-groups:") || strings.Contains(trimmed, "mixed-port:") {
		return "mihomo-yaml"
	}
	if looksLikeRawLinks(trimmed) {
		return "raw-links"
	}
	compact := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t', ' ':
			return -1
		default:
			return r
		}
	}, trimmed)
	if decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(compact))); err == nil && looksLikeRawLinks(string(decoded)) {
		return "base64-links"
	}
	return "unknown"
}

func looksLikeRawLinks(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "trojan://") || strings.HasPrefix(line, "hysteria2://") {
			return true
		}
	}
	return false
}

// CheckPortInUse checks if a TCP port is already in use.
func CheckPortInUse(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

// LookupHost resolves a hostname and returns the first IP address.
func LookupHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("未找到 %s 的解析地址", host)
	}
	return addrs[0], nil
}
