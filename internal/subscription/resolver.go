package subscription

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/system"

	"gopkg.in/yaml.v3"
)

// PlanKind identifies the resolved config style.
type PlanKind string

const (
	PlanKindStatic   PlanKind = "static"
	PlanKindYAML     PlanKind = "yaml"
	PlanKindProvider PlanKind = "provider"
)

// ResolvedConfigPlan is a write-ready Mihomo config plan.
type ResolvedConfigPlan struct {
	Kind            PlanKind
	ContentKind     string
	DetectedFormat  string
	Summary         string
	FetchDetail     string
	UsedProxyEnv    bool
	ProxyCount      int
	VerifyInventory bool
	Warnings        []string
	RemovedFields   []string
	Sanitized       bool
	MihomoConfig    *core.MihomoConfig
	RawYAML         []byte
}

// RenderYAML renders the plan to YAML.
func (p *ResolvedConfigPlan) RenderYAML() ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("配置计划为空")
	}
	if len(p.RawYAML) > 0 {
		return append([]byte{}, p.RawYAML...), nil
	}
	if p.MihomoConfig == nil {
		return nil, fmt.Errorf("未生成可写入的配置")
	}
	return core.RenderYAML(p.MihomoConfig)
}

// Save writes the plan to disk with backup and validation.
func (p *ResolvedConfigPlan) Save(path string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("配置计划为空")
	}
	if len(p.RawYAML) > 0 {
		return config.SaveRawYAML(p.RawYAML, path)
	}
	if p.MihomoConfig == nil {
		return "", fmt.Errorf("未生成可写入的配置")
	}
	return config.SaveMihomoConfig(p.MihomoConfig, path)
}

// Resolver resolves subscription inputs into write-ready config plans.
type Resolver struct {
	prepareURL func(string, time.Duration) (*system.PreparedSubscription, error)
}

// NewResolver creates a Resolver with the default remote fetcher.
func NewResolver() *Resolver {
	return &Resolver{
		prepareURL: system.PrepareSubscriptionURL,
	}
}

// ResolveRemoteURL resolves a remote subscription URL into a config plan.
func (r *Resolver) ResolveRemoteURL(cfg *core.AppConfig, rawURL string, timeout time.Duration) (*ResolvedConfigPlan, error) {
	prepared, err := r.prepareURL(rawURL, timeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = prepared.Cleanup()
	}()

	plan, err := r.ResolveContent(cfg, prepared.Body)
	if err != nil {
		contentKind := system.ProbeContentKind(prepared.Body)
		if contentKind == "unknown" && looksLikeProviderConfig(prepared.Body) {
			providerPlan, providerErr := resolveProviderConfigPlan(cfg, prepared.Body, rawURL)
			if providerErr != nil {
				return nil, providerErr
			}
			providerPlan.FetchDetail = prepared.FetchDetail
			providerPlan.UsedProxyEnv = system.HasProxyEnvForDisplay()
			return providerPlan, nil
		}
		if contentKind == "html" || contentKind == "empty" {
			return nil, fmt.Errorf("订阅返回了不可用内容 (%s): %s", contentKind, previewSubscriptionBody(prepared.Body))
		}
		if contentKind == "unknown" {
			return nil, fmt.Errorf("订阅返回了无法识别的内容: %s", previewSubscriptionBody(prepared.Body))
		}
		return nil, err
	}

	plan.FetchDetail = prepared.FetchDetail
	plan.UsedProxyEnv = system.HasProxyEnvForDisplay()
	return plan, nil
}

// ResolveContent resolves raw subscription content into a config plan.
func (r *Resolver) ResolveContent(cfg *core.AppConfig, content []byte) (*ResolvedConfigPlan, error) {
	contentKind := system.ProbeContentKind(content)
	switch contentKind {
	case "raw-links", "base64-links":
		parsed, err := Parse(content)
		if err != nil {
			return nil, err
		}
		return &ResolvedConfigPlan{
			Kind:            PlanKindStatic,
			ContentKind:     contentKind,
			DetectedFormat:  parsed.DetectedFormat,
			Summary:         fmt.Sprintf("已解析 %d 个节点，使用静态配置", len(parsed.Names)),
			ProxyCount:      len(parsed.Names),
			VerifyInventory: true,
			MihomoConfig:    core.BuildStaticMihomoConfig(cfg, parsed.Proxies, parsed.Names),
		}, nil
	case "mihomo-yaml":
		patched, err := PatchRemoteYAML(content, cfg)
		if err != nil {
			return nil, err
		}
		return &ResolvedConfigPlan{
			Kind:            PlanKindYAML,
			ContentKind:     contentKind,
			DetectedFormat:  contentKind,
			Summary:         "检测到 Mihomo/Clash YAML，已转为本地静态配置",
			VerifyInventory: true,
			Warnings:        patched.Warnings,
			RemovedFields:   patched.RemovedFields,
			Sanitized:       patched.Sanitized,
			RawYAML:         patched.YAML,
		}, nil
	default:
		return nil, fmt.Errorf("未识别的订阅内容格式: %s", contentKind)
	}
}

func previewSubscriptionBody(body []byte) string {
	preview := strings.TrimSpace(string(body))
	if preview == "" {
		return "空响应"
	}
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	return preview
}

func looksLikeProviderConfig(body []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return false
	}
	_, hasProviders := doc["proxy-providers"]
	_, hasProxies := doc["proxies"]
	_, hasGroups := doc["proxy-groups"]
	return hasProviders && !hasProxies && !hasGroups
}

func resolveProviderConfigPlan(cfg *core.AppConfig, body []byte, rawURL string) (*ResolvedConfigPlan, error) {
	patched, err := PatchRemoteYAML(body, cfg)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if err := yaml.Unmarshal(patched.YAML, &doc); err != nil {
		return nil, fmt.Errorf("解析 provider 配置失败: %w", err)
	}

	base := core.BuildMihomoConfig(cfg)
	doc["mode"] = base.Mode
	doc["dns"] = base.DNS
	if _, ok := doc["rules"]; !ok || len(asYAMLList(doc["rules"])) == 0 {
		doc["rules"] = append([]string{}, base.Rules...)
	}

	providers, ok := doc["proxy-providers"].(map[string]any)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("provider 配置缺少可用的 proxy-providers")
	}

	if _, ok := doc["proxy-groups"]; !ok || len(asYAMLList(doc["proxy-groups"])) == 0 {
		doc["proxy-groups"] = []any{map[string]any{
			"name": "PROXY",
			"type": "select",
			"use":  sortedProviderNames(providers),
		}}
		patched.Warnings = append(patched.Warnings, "provider 配置未声明 proxy-groups，已补全默认 PROXY 组")
	}

	rendered, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("写回 provider 配置失败: %w", err)
	}

	patched.Warnings = dedupeStrings(patched.Warnings)
	return &ResolvedConfigPlan{
		Kind:            PlanKindProvider,
		ContentKind:     "provider-yaml",
		DetectedFormat:  "provider-yaml",
		Summary:         fmt.Sprintf("检测到 provider 配置，已保留远程 provider 并补全本地运行默认项: %s", rawURL),
		VerifyInventory: true,
		Warnings:        patched.Warnings,
		RemovedFields:   patched.RemovedFields,
		Sanitized:       patched.Sanitized,
		RawYAML:         rendered,
	}, nil
}

func asYAMLList(value any) []any {
	list, _ := value.([]any)
	return list
}

func sortedProviderNames(providers map[string]any) []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
