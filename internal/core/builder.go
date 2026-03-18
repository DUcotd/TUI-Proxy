// Package core provides configuration building logic.
package core

// BuildMihomoConfig generates a MihomoConfig from the given AppConfig.
func BuildMihomoConfig(cfg *AppConfig) *MihomoConfig {
	m := &MihomoConfig{
		MixedPort:         cfg.MixedPort,
		AllowLan:          false,
		Mode:              "rule",
		LogLevel:          "info",
		ExternalController: cfg.ControllerAddr,
		ProxyProviders: map[string]*ProxyProvider{
			"airport": {
				Type:     "http",
				URL:      cfg.SubscriptionURL,
				Path:     cfg.ProviderPath,
				Interval: 3600,
				Override: &Override{
					SkipCertVerify: true,
				},
				HealthCheck: &HealthCheck{
					Enable:   cfg.EnableHealthCheck,
					URL:      "https://cp.cloudflare.com/",
					Interval: 300,
				},
			},
		},
		ProxyGroups: []*ProxyGroup{
			{
				Name: "PROXY",
				Type: "select",
				Use:  []string{"airport"},
			},
			{
				Name:     "auto",
				Type:     "url-test",
				Use:      []string{"airport"},
				URL:      "https://cp.cloudflare.com/",
				Interval: 300,
			},
			{
				Name:     "fallback",
				Type:     "fallback",
				Use:      []string{"airport"},
				URL:      "https://cp.cloudflare.com/",
				Interval: 300,
			},
		},
		DNS: &DNSConfig{
			Enable:       true,
			IPv6:         false,
			EnhancedMode: "fake-ip",
			FakeIPRange:  "198.18.0.1/16",
			NameServer: []string{
				"https://1.1.1.1/dns-query",
				"https://dns.google/dns-query",
			},
			Fallback: []string{
				"https://1.1.1.1/dns-query",
				"https://dns.google/dns-query",
				"tls://8.8.4.4:853",
			},
			DefaultNameserver: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			DirectNameserver: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
		},
		Rules: []string{
			// Local/lan traffic
			"DOMAIN-SUFFIX,local,DIRECT",
			"IP-CIDR,127.0.0.0/8,DIRECT",
			"IP-CIDR,172.16.0.0/12,DIRECT",
			"IP-CIDR,192.168.0.0/16,DIRECT",
			"IP-CIDR,10.0.0.0/8,DIRECT",
			"IP-CIDR,100.64.0.0/10,DIRECT",
			// China mainland - direct
			"GEOSITE,cn,DIRECT",
			"GEOIP,CN,DIRECT",
			// Fallback
			"MATCH,PROXY",
		},
	}

	// Add TUN config only in TUN mode
	if cfg.Mode == "tun" {
		m.TUN = &TUNConfig{
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

	return m
}
