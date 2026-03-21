package subscription

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"clashctl/internal/core"
	"clashctl/internal/netsec"
)

// PatchRemoteYAML applies server-friendly defaults to a downloaded Clash/Mihomo YAML profile.
func PatchRemoteYAML(data []byte, cfg *core.AppConfig) ([]byte, error) {
	// First validate security of the YAML content
	warnings, err := ValidateYAMLSecurity(data, false)
	if err != nil {
		// Try to sanitize instead of failing completely
		sanitized, removed, sanitizeErr := SanitizeYAML(data)
		if sanitizeErr != nil {
			return nil, fmt.Errorf("订阅 YAML 安全校验失败: %w", err)
		}
		// Use sanitized version and add warning
		data = sanitized
		_ = removed // Could log removed fields
	}
	_ = warnings // Could log warnings

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析订阅 YAML 失败: %w", err)
	}

	patchedDoc := sanitizeRemoteYAMLDocument(doc, cfg)

	patched, err := yaml.Marshal(patchedDoc)
	if err != nil {
		return nil, fmt.Errorf("写回订阅 YAML 失败: %w", err)
	}
	return patched, nil
}

func sanitizeRemoteYAMLDocument(doc map[string]any, cfg *core.AppConfig) map[string]any {
	patched := map[string]any{}
	for key, value := range doc {
		lowerKey := strings.ToLower(key)
		if !allowedTopLevelFields[lowerKey] {
			continue
		}

		switch lowerKey {
		case "proxies":
			if proxies := sanitizeProxyList(value); len(proxies) > 0 {
				patched[key] = proxies
			}
		case "proxy-providers":
			if providers := sanitizeProxyProviders(value, cfg); len(providers) > 0 {
				patched[key] = providers
			}
		case "proxy-groups":
			if groups := sanitizeProxyGroups(value); len(groups) > 0 {
				patched[key] = groups
			}
		case "rules":
			if rules := sanitizeRules(value); len(rules) > 0 {
				patched[key] = rules
			}
		default:
			patched[key] = cloneYAMLValue(value)
		}
	}

	patched["allow-lan"] = false
	patched["external-controller"] = cfg.ControllerAddr
	patched["log-level"] = "info"
	if cfg.Mode == "mixed" {
		patched["mixed-port"] = cfg.MixedPort
	}

	return patched
}

func sanitizeProxyList(value any) []any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(list))
	for _, entry := range list {
		proxy, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, cloneYAMLValue(proxy))
	}
	return out
}

func sanitizeProxyProviders(value any, cfg *core.AppConfig) map[string]any {
	providers, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]any, len(providers))
	for name, entry := range providers {
		provider, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		cleaned := sanitizeProxyProvider(name, provider, cfg)
		if len(cleaned) > 0 {
			out[name] = cleaned
		}
	}
	return out
}

func sanitizeProxyProvider(name string, provider map[string]any, cfg *core.AppConfig) map[string]any {
	out := map[string]any{}
	if providerType, ok := provider["type"].(string); ok && providerType == "http" {
		out["type"] = providerType
	} else {
		return nil
	}

	rawURL, ok := provider["url"].(string)
	if !ok {
		return nil
	}
	if _, err := netsec.ValidateRemoteHTTPURL(rawURL, netsec.URLValidationOptions{}); err != nil {
		return nil
	}
	out["url"] = rawURL
	out["path"] = filepath.Join(cfg.ConfigDir, "providers", sanitizePathSegment(name)+".yaml")

	if interval, ok := asPositiveInt(provider["interval"]); ok {
		out["interval"] = interval
	}
	if filter, ok := provider["filter"].(string); ok && strings.TrimSpace(filter) != "" {
		out["filter"] = filter
	}
	if excludeFilter, ok := provider["exclude-filter"].(string); ok && strings.TrimSpace(excludeFilter) != "" {
		out["exclude-filter"] = excludeFilter
	}
	if healthCheck := sanitizeHealthCheck(provider["health-check"]); len(healthCheck) > 0 {
		out["health-check"] = healthCheck
	}

	return out
}

func sanitizeProxyGroups(value any) []any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	allowedKeys := map[string]bool{
		"name":      true,
		"type":      true,
		"proxies":   true,
		"use":       true,
		"url":       true,
		"interval":  true,
		"lazy":      true,
		"tolerance": true,
		"filter":    true,
	}
	out := make([]any, 0, len(list))
	for _, entry := range list {
		group, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		cleaned := map[string]any{}
		for key, groupValue := range group {
			if !allowedKeys[strings.ToLower(key)] {
				continue
			}
			cleaned[key] = cloneYAMLValue(groupValue)
		}
		if len(cleaned) > 0 {
			out = append(out, cleaned)
		}
	}
	return out
}

func sanitizeRules(value any) []any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(list))
	for _, entry := range list {
		rule, ok := entry.(string)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(rule), "script") {
			continue
		}
		out = append(out, rule)
	}
	return out
}

func sanitizeHealthCheck(value any) map[string]any {
	healthCheck, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if enabled, ok := healthCheck["enable"].(bool); ok {
		out["enable"] = enabled
	}
	if urlValue, ok := healthCheck["url"].(string); ok {
		if _, err := netsec.ValidateRemoteHTTPURL(urlValue, netsec.URLValidationOptions{}); err == nil {
			out["url"] = urlValue
		}
	}
	if interval, ok := asPositiveInt(healthCheck["interval"]); ok {
		out["interval"] = interval
	}
	if lazy, ok := healthCheck["lazy"].(bool); ok {
		out["lazy"] = lazy
	}
	return out
}

func cloneYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = cloneYAMLValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneYAMLValue(item))
		}
		return out
	default:
		return typed
	}
}

func sanitizePathSegment(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "provider"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "provider"
	}
	return cleaned
}

func asPositiveInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, typed > 0
	case int64:
		return int(typed), typed > 0
	case float64:
		return int(typed), typed > 0 && float64(int(typed)) == typed
	default:
		return 0, false
	}
}
