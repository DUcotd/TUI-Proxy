package subscription

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"clashctl/internal/core"
)

// PatchRemoteYAML applies server-friendly defaults to a downloaded Clash/Mihomo YAML profile.
func PatchRemoteYAML(data []byte, cfg *core.AppConfig) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析订阅 YAML 失败: %w", err)
	}

	doc["allow-lan"] = false
	doc["external-controller"] = cfg.ControllerAddr
	doc["log-level"] = "info"
	if cfg.Mode == "mixed" {
		doc["mixed-port"] = cfg.MixedPort
	}

	patched, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("写回订阅 YAML 失败: %w", err)
	}
	return patched, nil
}
