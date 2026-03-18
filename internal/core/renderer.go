// Package core provides YAML serialization for Mihomo configurations.
package core

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// RenderYAML serializes a MihomoConfig to a YAML byte slice.
func RenderYAML(cfg *MihomoConfig) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("YAML 序列化失败: %w", err)
	}
	return data, nil
}
