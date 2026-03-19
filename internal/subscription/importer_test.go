package subscription

import "testing"

func TestParseRawLinks(t *testing.T) {
	input := []byte("vless://uuid@example.com:443?type=tcp&security=reality&sni=music.apple.com&pbk=pub&sid=short#node-a\n" +
		"trojan://pass@host.example:443?type=ws&host=cdn.example&path=%2Fws#node-b\n" +
		"hysteria2://pwd@hy.example:8443/?insecure=1&sni=files.example#node-c\n")

	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.DetectedFormat != "raw-links" {
		t.Fatalf("DetectedFormat = %q, want raw-links", parsed.DetectedFormat)
	}
	if len(parsed.Proxies) != 3 {
		t.Fatalf("len(Proxies) = %d, want 3", len(parsed.Proxies))
	}
	if parsed.Names[0] != "node-a" || parsed.Names[1] != "node-b" || parsed.Names[2] != "node-c" {
		t.Fatalf("Names = %#v", parsed.Names)
	}
	if got := parsed.Proxies[0]["type"]; got != "vless" {
		t.Fatalf("proxy[0].type = %v, want vless", got)
	}
	if got := parsed.Proxies[1]["type"]; got != "trojan" {
		t.Fatalf("proxy[1].type = %v, want trojan", got)
	}
	if got := parsed.Proxies[2]["type"]; got != "hysteria2" {
		t.Fatalf("proxy[2].type = %v, want hysteria2", got)
	}
	if got := parsed.Proxies[1]["network"]; got != "ws" {
		t.Fatalf("proxy[1].network = %v, want ws", got)
	}
}

func TestParseBase64Links(t *testing.T) {
	input := []byte("dmxlc3M6Ly91dWlkQGV4YW1wbGUuY29tOjQ0Mz90eXBlPXRjcCZzZWN1cml0eT1yZWFsaXR5JnNuaT1tdXNpYy5hcHBsZS5jb20mcGJrPXB1YiZzaWQ9c2hvcnQjbm9kZS1h")

	parsed, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.DetectedFormat != "base64-links" {
		t.Fatalf("DetectedFormat = %q, want base64-links", parsed.DetectedFormat)
	}
	if len(parsed.Proxies) != 1 {
		t.Fatalf("len(Proxies) = %d, want 1", len(parsed.Proxies))
	}
}
