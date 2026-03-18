// Package system provides network utility functions.
package system

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// CheckURLReachable performs an HTTP GET request to verify a URL is accessible.
// Uses GET instead of HEAD because many subscription servers don't support HEAD.
func CheckURLReachable(rawURL string, timeout time.Duration) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("无法构建请求: %w", err)
	}
	// Some providers require a User-Agent to return proper content
	req.Header.Set("User-Agent", "clashctl/2.1.4")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("无法访问 %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s 返回 HTTP %d", rawURL, resp.StatusCode)
	}
	return nil
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
		return "", fmt.Errorf("no addresses found for %s", host)
	}
	return addrs[0], nil
}
