package cmd

import (
	"os"
	"strings"
	"testing"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
)

func TestRunConfigShowRejectsOversizedConfig(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := core.DefaultAppConfig()
	cfg.ConfigDir = home
	cfg.SubscriptionURL = "https://example.com/sub"
	if err := app.SaveAppConfig(cfg); err != nil {
		t.Fatalf("SaveAppConfig() error = %v", err)
	}

	oversized := strings.Repeat("a", config.MaxConfigFileSize+1)
	if err := os.WriteFile(mihomoConfigPath(cfg), []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runConfigShow(nil, nil)
	if err == nil {
		t.Fatal("runConfigShow() should reject oversized config files")
	}
	if !strings.Contains(err.Error(), "配置文件过大") {
		t.Fatalf("runConfigShow() error = %v", err)
	}
}
