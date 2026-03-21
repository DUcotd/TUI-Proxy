package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateCommandProvidesSelfAlias(t *testing.T) {
	if !hasAlias(updateCmd, "self") {
		t.Fatal("update command should provide the self alias")
	}
}

func TestFinishUpdateReportStoresError(t *testing.T) {
	prev := updateJSON
	t.Cleanup(func() { updateJSON = prev })
	updateJSON = false
	report := &updateRunReport{CurrentVersion: "v1.0.0", Action: "check"}
	err := finishUpdateReport(report, errors.New("boom"))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("finishUpdateReport() error = %v", err)
	}
	if report.Error != "boom" {
		t.Fatalf("report.Error = %q, want boom", report.Error)
	}
}

func TestValidateDownloadedClashctlBinary(t *testing.T) {
	tmp := t.TempDir()
	good := filepath.Join(tmp, "clashctl-good")
	silent := filepath.Join(tmp, "clashctl-silent")
	broken := filepath.Join(tmp, "clashctl-broken")

	writeExecutableFile(t, good, "#!/bin/sh\necho 'clashctl v9.9.9'\n")
	writeExecutableFile(t, silent, "#!/bin/sh\nexit 0\n")
	writeExecutableFile(t, broken, "#!/bin/sh\necho 'boom' >&2\nexit 1\n")

	if err := validateDownloadedClashctlBinary(good); err != nil {
		t.Fatalf("validateDownloadedClashctlBinary(good) error = %v", err)
	}
	if err := validateDownloadedClashctlBinary(silent); err == nil {
		t.Fatal("validateDownloadedClashctlBinary(silent) should fail")
	}
	if err := validateDownloadedClashctlBinary(broken); err == nil {
		t.Fatal("validateDownloadedClashctlBinary(broken) should fail")
	}
}

func writeExecutableFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
