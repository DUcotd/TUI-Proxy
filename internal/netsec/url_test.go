package netsec

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestValidateRemoteHTTPURLRejectsLocalTargets(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/sub",
		"https://localhost/sub",
		"https://192.168.1.10/sub",
		"https://[::1]/sub",
	}

	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := ValidateRemoteHTTPURL(rawURL, URLValidationOptions{}); err == nil {
				t.Fatalf("ValidateRemoteHTTPURL(%q) should reject local target", rawURL)
			}
		})
	}
}

func TestValidateRemoteHTTPURLAllowsLocalTargetsWhenRequested(t *testing.T) {
	got, err := ValidateRemoteHTTPURL("http://127.0.0.1/sub", URLValidationOptions{AllowLocal: true})
	if err != nil {
		t.Fatalf("ValidateRemoteHTTPURL() error = %v", err)
	}
	if got.Hostname() != "127.0.0.1" {
		t.Fatalf("hostname = %q, want 127.0.0.1", got.Hostname())
	}
}

func TestValidateRemoteHTTPURLAllowsLocalTargetsWithEnvOverride(t *testing.T) {
	t.Setenv(localSubscriptionOverrideEnv, "true")

	got, err := ValidateRemoteHTTPURL("https://localhost/sub", URLValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateRemoteHTTPURL() error = %v", err)
	}
	if got.Hostname() != "localhost" {
		t.Fatalf("hostname = %q, want localhost", got.Hostname())
	}
}

func TestValidateRemoteHTTPURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := ValidateRemoteHTTPURL("ftp://example.com/sub", URLValidationOptions{}); err == nil {
		t.Fatal("ValidateRemoteHTTPURL() should reject unsupported schemes")
	}
}

func TestValidateRemoteHTTPURLRejectsResolutionFailuresWhenRequested(t *testing.T) {
	prev := lookupIPAddr
	lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
		return nil, fmt.Errorf("dns timeout")
	}
	defer func() {
		lookupIPAddr = prev
	}()

	if _, err := ValidateRemoteHTTPURL("https://example.com/sub", URLValidationOptions{ResolveHost: true}); err == nil {
		t.Fatal("ValidateRemoteHTTPURL() should reject resolution failures")
	}
}
