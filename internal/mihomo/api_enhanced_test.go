package mihomo

import "testing"

func TestNormalizeProxyType(t *testing.T) {
	tests := map[string]string{
		"Selector":     "select",
		"URLTest":      "url-test",
		"Fallback":     "fallback",
		"load-balance": "load-balance",
		"Compatible":   "compatible",
	}

	for in, want := range tests {
		if got := NormalizeProxyType(in); got != want {
			t.Fatalf("NormalizeProxyType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsProxyGroupType(t *testing.T) {
	if !IsProxyGroupType("Selector") || !IsProxyGroupType("URLTest") {
		t.Fatal("expected Selector and URLTest to be treated as groups")
	}
	if IsProxyGroupType("Vless") || IsProxyGroupType("Trojan") {
		t.Fatal("expected node protocols not to be treated as groups")
	}
}
