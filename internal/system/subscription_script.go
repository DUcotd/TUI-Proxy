package system

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"clashctl/internal/core"
)

//go:embed scripts/prepare-subscription.sh
var prepareSubscriptionScript string

type PreparedSubscription struct {
	Body        []byte
	ContentPath string
	InfoPath    string
	FetchDetail string
}

// PrepareSubscriptionURL downloads a subscription URL via the bundled shell helper.
func PrepareSubscriptionURL(rawURL string, timeout time.Duration) (*PreparedSubscription, error) {
	workDir, err := os.MkdirTemp("", "clashctl-sub-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	scriptPath := filepath.Join(workDir, "prepare-subscription.sh")
	if err := os.WriteFile(scriptPath, []byte(prepareSubscriptionScript), 0700); err != nil {
		return nil, fmt.Errorf("写入订阅脚本失败: %w", err)
	}

	outDir := filepath.Join(workDir, "output")
	cmd := exec.Command("/bin/sh", scriptPath, rawURL, outDir, fmt.Sprintf("%d", int(timeout.Seconds())), "clashctl/"+core.AppVersion)
	cmd.Env = StripProxyEnv(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("订阅脚本执行失败: %s", msg)
	}

	contentPath := strings.TrimSpace(string(output))
	if contentPath == "" {
		contentPath = filepath.Join(outDir, "subscription.txt")
	}
	body, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("读取订阅内容失败: %w", err)
	}
	infoPath := filepath.Join(outDir, "subscription.info")
	infoData, _ := os.ReadFile(infoPath)

	return &PreparedSubscription{
		Body:        body,
		ContentPath: contentPath,
		InfoPath:    infoPath,
		FetchDetail: strings.TrimSpace(string(infoData)),
	}, nil
}
