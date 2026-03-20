package app

import (
	"strings"
	"testing"
)

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string // Should contain these after sanitization
		notContains []string // Should NOT contain these
	}{
		{
			name:        "URL sanitization",
			input:       "Fetching https://example.com/sub/path?token=secret",
			contains:    []string{"https://example.com"},
			notContains: []string{"secret", "token"},
		},
		{
			name:        "UUID sanitization",
			input:       "UUID: 550e8400-e29b-41d4-a716-446655440000",
			contains:    []string{"[UUID_REDACTED]"},
			notContains: []string{"550e8400"},
		},
		{
			name:        "password sanitization",
			input:       "password=mysecretpassword",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"mysecretpassword"},
		},
		{
			name:        "GitHub token sanitization",
			input:       "token=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"ghp_1234567890"},
		},
		{
			name:        "Bearer token sanitization",
			input:       "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForLog(tt.input)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("sanitizeForLog(%q) = %q, want to contain %q", tt.input, result, s)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(result, s) {
					t.Errorf("sanitizeForLog(%q) = %q, should NOT contain %q", tt.input, result, s)
				}
			}
		})
	}
}
