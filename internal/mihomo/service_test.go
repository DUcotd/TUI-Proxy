package mihomo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderServiceFileQuotesPathsAndIncludesIdentity(t *testing.T) {
	data, err := renderServiceFile(ServiceConfig{
		Binary:      "/usr/local/bin/mihomo beta",
		ConfigDir:   "/etc/mihomo/test dir",
		ServiceName: DefaultServiceName,
		User:        "mihomo",
		Group:       "mihomo",
	})
	if err != nil {
		t.Fatalf("renderServiceFile() error = %v", err)
	}

	text := string(data)
	for _, want := range []string{
		`ExecStart="/usr/local/bin/mihomo beta" -d "/etc/mihomo/test dir"`,
		"User=mihomo",
		"Group=mihomo",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("service file missing %q in:\n%s", want, text)
		}
	}
}

func TestRenderServiceFileOmitsIdentityWhenUnset(t *testing.T) {
	data, err := renderServiceFile(ServiceConfig{
		Binary:      "/usr/local/bin/mihomo",
		ConfigDir:   "/etc/mihomo",
		ServiceName: DefaultServiceName,
	})
	if err != nil {
		t.Fatalf("renderServiceFile() error = %v", err)
	}

	text := string(data)
	if strings.Contains(text, "User=") || strings.Contains(text, "Group=") {
		t.Fatalf("service file should omit identity when unset:\n%s", text)
	}
}

func TestWriteFileAtomicCreatesFinalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "systemd", "clashctl-mihomo.service")
	want := []byte("[Unit]\nDescription=test\n")

	if err := writeFileAtomic(path, want, 0644); err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("file content = %q, want %q", string(got), string(want))
	}

	matches, err := filepath.Glob(path + ".tmp-*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files should be cleaned up, found %v", matches)
	}
}
