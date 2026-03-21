package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPreparedSubscriptionBodyRejectsOversizeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscription.txt")
	data := make([]byte, MaxPreparedSubscriptionBytes+1)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := readPreparedSubscriptionBody(path)
	if err == nil {
		t.Fatal("readPreparedSubscriptionBody() should reject oversized files")
	}
	if !strings.Contains(err.Error(), "过大") {
		t.Fatalf("readPreparedSubscriptionBody() error = %v, want size hint", err)
	}
}

func TestPreparedSubscriptionCleanupRemovesTempDir(t *testing.T) {
	dir := t.TempDir()
	prepared := &PreparedSubscription{TempDir: dir}

	if err := prepared.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("TempDir should be removed, stat err = %v", err)
	}
}
