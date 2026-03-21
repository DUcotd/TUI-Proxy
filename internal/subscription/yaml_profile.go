package subscription

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"clashctl/internal/core"
	"clashctl/internal/netsec"
)

type PatchedYAMLResult struct {
	YAML          []byte
	Warnings      []string
	RemovedFields []string
	Sanitized     bool
}

// PatchRemoteYAML applies server-friendly defaults to a downloaded Clash/Mihomo YAML profile.
func PatchRemoteYAML(data []byte, cfg *core.AppConfig) (*PatchedYAMLResult, error) {
	// First validate security of the YAML content
	warnings, err := ValidateYAMLSecurity(data, false)
	var removed []string
	sanitized := false
	if err != nil {
		// Try to sanitize instead of failing completely
		sanitizedYAML, sanitizedRemoved, sanitizeErr := SanitizeYAML(data)
		if sanitizeErr != nil {
			return nil, fmt.Errorf("订阅 YAML 安全校验失败: %w", err)
		}
		// Use sanitized version and add warning
		data = sanitizedYAML
		removed = append(removed, sanitizedRemoved...)
		if len(sanitizedRemoved) > 0 {
			warnings = append(warnings, fmt.Sprintf("已移除高风险字段: %s", strings.Join(sanitizedRemoved, ", ")))
		}
		sanitized = true
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析订阅 YAML 失败: %w", err)
	}

	patchedDoc, patchedRemoved := sanitizeRemoteYAMLDocument(doc, cfg)
	removed = append(removed, patchedRemoved...)
	if len(patchedRemoved) > 0 {
		sanitized = true
		warnings = append(warnings, fmt.Sprintf("已裁剪不兼容字段: %s", strings.Join(patchedRemoved, ", ")))
	}

	patched, err := yaml.Marshal(patchedDoc)
	if err != nil {
		return nil, fmt.Errorf("写回订阅 YAML 失败: %w", err)
	}
	return &PatchedYAMLResult{
		YAML:          patched,
		Warnings:      dedupeStrings(warnings),
		RemovedFields: dedupeStrings(removed),
		Sanitized:     sanitized,
	}, nil
}

func sanitizeRemoteYAMLDocument(doc map[string]any, cfg *core.AppConfig) (map[string]any, []string) {
	patched := map[string]any{}
	var removed []string
	for key, value := range doc {
		lowerKey := strings.ToLower(key)
		if !allowedTopLevelFields[lowerKey] {
			removed = append(removed, key)
			continue
		}

		switch lowerKey {
		case "mixed-port":
			continue
		case "proxies":
			if proxies := sanitizeProxyList(value); len(proxies) > 0 {
				patched[key] = proxies
			} else {
				removed = append(removed, key)
			}
		case "proxy-providers":
			providers, providerRemoved := sanitizeProxyProviders(value, cfg)
			if len(providers) > 0 {
				patched[key] = providers
				removed = append(removed, providerRemoved...)
			} else {
				removed = append(removed, append([]string{key}, providerRemoved...)...)
			}
		case "proxy-groups":
			if groups := sanitizeProxyGroups(value); len(groups) > 0 {
				patched[key] = groups
			} else {
				removed = append(removed, key)
			}
		case "rules":
			if rules := sanitizeRules(value); len(rules) > 0 {
				patched[key] = rules
			} else {
				removed = append(removed, key)
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
	} else {
		if _, hadMixedPort := doc["mixed-port"]; hadMixedPort {
			removed = append(removed, "mixed-port")
		}
		patched["tun"] = buildPatchedTUNConfig()
	}

	return patched, dedupeStrings(removed)
}

func buildPatchedTUNConfig() *core.TUNConfig {
	return &core.TUNConfig{
		Enable:              true,
		Stack:               "mixed",
		AutoRoute:           true,
		AutoRedirect:        true,
		AutoDetectInterface: true,
		DNSHijack: []string{
			"any:53",
			"tcp://any:53",
		},
	}
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

func sanitizeProxyProviders(value any, cfg *core.AppConfig) (map[string]any, []string) {
	providers, ok := value.(map[string]any)
	if !ok {
		return nil, nil
	}
	out := make(map[string]any, len(providers))
	var removed []string
	for name, entry := range providers {
		provider, ok := entry.(map[string]any)
		if !ok {
			removed = append(removed, "proxy-providers."+name)
			continue
		}
		cleaned, providerRemoved := sanitizeProxyProvider(name, provider, cfg)
		if len(cleaned) > 0 {
			out[name] = cleaned
		} else {
			removed = append(removed, "proxy-providers."+name)
		}
		removed = append(removed, providerRemoved...)
	}
	return out, dedupeStrings(removed)
}

func sanitizeProxyProvider(name string, provider map[string]any, cfg *core.AppConfig) (map[string]any, []string) {
	out := map[string]any{}
	var removed []string
	if providerType, ok := provider["type"].(string); ok && providerType == "http" {
		out["type"] = providerType
	} else {
		return nil, []string{"proxy-providers." + name + ".type"}
	}

	rawURL, ok := provider["url"].(string)
	if !ok {
		return nil, []string{"proxy-providers." + name + ".url"}
	}
	if _, err := netsec.ValidateRemoteHTTPURL(rawURL, netsec.URLValidationOptions{}); err != nil {
		return nil, []string{"proxy-providers." + name + ".url"}
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
	healthCheck, healthRemoved := sanitizeHealthCheck(provider["health-check"], "proxy-providers."+name+".health-check")
	if len(healthCheck) > 0 {
		out["health-check"] = healthCheck
		removed = append(removed, healthRemoved...)
	} else if provider["health-check"] != nil {
		removed = append(removed, append([]string{"proxy-providers." + name + ".health-check"}, healthRemoved...)...)
	}

	for key := range provider {
		switch strings.ToLower(key) {
		case "type", "url", "interval", "filter", "exclude-filter", "health-check":
		default:
			removed = append(removed, "proxy-providers."+name+"."+key)
		}
	}

	return out, dedupeStrings(removed)
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

func sanitizeHealthCheck(value any, fieldPath string) (map[string]any, []string) {
	healthCheck, ok := value.(map[string]any)
	if !ok {
		return nil, nil
	}
	out := map[string]any{}
	var removed []string
	if enabled, ok := healthCheck["enable"].(bool); ok {
		out["enable"] = enabled
	}
	if urlValue, ok := healthCheck["url"].(string); ok {
		if _, err := netsec.ValidateRemoteHTTPURL(urlValue, netsec.URLValidationOptions{}); err == nil {
			out["url"] = urlValue
		} else {
			removed = append(removed, fieldPath+".url")
		}
	}
	if interval, ok := asPositiveInt(healthCheck["interval"]); ok {
		out["interval"] = interval
	}
	if lazy, ok := healthCheck["lazy"].(bool); ok {
		out["lazy"] = lazy
	}
	for key := range healthCheck {
		switch strings.ToLower(key) {
		case "enable", "url", "interval", "lazy":
		default:
			removed = append(removed, fieldPath+"."+key)
		}
	}
	return out, dedupeStrings(removed)
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

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	slices.Sort(values)
	return slices.Compact(values)
}
