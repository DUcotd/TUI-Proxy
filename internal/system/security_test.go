package system

import (
	"strings"
	"testing"
)

func TestValidateOutputPath_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "safe relative path",
			path:    "config.yaml",
			wantErr: false,
		},
		{
			name:    "safe absolute path in mihomo dir",
			path:    "/etc/mihomo/config.yaml",
			wantErr: false,
		},
		{
			name:    "safe tmp path",
			path:    "/tmp/config.yaml",
			wantErr: false,
		},
		{
			name:    "path traversal with ../",
			path:    "../../etc/passwd",
			wantErr: true,
			errMsg:  "路径遍历",
		},
		{
			name:    "dangerous /etc/passwd",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "dangerous /etc/shadow",
			path:    "/etc/shadow",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "dangerous /boot",
			path:    "/boot/config",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "dangerous /bin",
			path:    "/bin/malicious",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "dangerous /usr/bin",
			path:    "/usr/bin/malicious",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "dangerous /etc/crontab",
			path:    "/etc/crontab",
			wantErr: true,
			errMsg:  "系统路径",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "不能为空",
		},
		{
			name:    "path traversal encoded",
			path:    "/etc/mihomo/../../../etc/passwd",
			wantErr: true,
			errMsg:  "路径遍历",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateOutputPath(%q) error = %v, want error containing %q", tt.path, err, tt.errMsg)
			}
		})
	}
}

func TestValidateSubscriptionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com/sub",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/sub",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "file protocol",
			url:     "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "URL with semicolon injection",
			url:     "https://example.com/sub;rm -rf /",
			wantErr: true,
		},
		{
			name:    "URL with pipe injection",
			url:     "https://example.com/sub|curl evil.com",
			wantErr: true,
		},
		{
			name:    "URL with backtick injection",
			url:     "https://example.com/sub`whoami`",
			wantErr: true,
		},
		{
			name:    "URL with command substitution",
			url:     "https://example.com/sub$(whoami)",
			wantErr: true,
		},
		{
			name:    "URL with && operator",
			url:     "https://example.com/sub&&whoami",
			wantErr: true,
		},
		{
			name:    "URL with || operator",
			url:     "https://example.com/sub||whoami",
			wantErr: true,
		},
		{
			name:    "URL with newline",
			url:     "https://example.com/sub\nrm -rf /",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubscriptionURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubscriptionURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
