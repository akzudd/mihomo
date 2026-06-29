package provider

import (
	"testing"

	"github.com/metacubex/mihomo/common/yaml"
)

func TestNewProxiesParserAppliesOverrideToDialerProxy(t *testing.T) {
	prefix := "pre-"
	suffix := "-suf"

	tests := []struct {
		name                string
		providerDialerProxy string
		proxyDialerProxy    string
		additionalPrefix    *string
		additionalSuffix    *string
		expectedProxyName   string
		expectedDialerProxy string
	}{
		{
			name:                "provider-level dialer-proxy",
			providerDialerProxy: "base",
			additionalPrefix:    &prefix,
			additionalSuffix:    &suffix,
			expectedProxyName:   "pre-node-suf",
			expectedDialerProxy: "pre-base-suf",
		},
		{
			name:                "proxy-level dialer-proxy",
			proxyDialerProxy:    "base",
			additionalPrefix:    &prefix,
			additionalSuffix:    &suffix,
			expectedProxyName:   "pre-node-suf",
			expectedDialerProxy: "pre-base-suf",
		},
		{
			name:                "proxy-level dialer-proxy overrides provider-level dialer-proxy",
			providerDialerProxy: "provider-base",
			proxyDialerProxy:    "proxy-base",
			additionalPrefix:    &prefix,
			additionalSuffix:    &suffix,
			expectedDialerProxy: "pre-proxy-base-suf",
			expectedProxyName:   "pre-node-suf",
		},
		{
			name:                "prefix only",
			providerDialerProxy: "base",
			additionalPrefix:    &prefix,
			expectedProxyName:   "pre-node",
			expectedDialerProxy: "pre-base",
		},
		{
			name:                "suffix only",
			providerDialerProxy: "base",
			additionalSuffix:    &suffix,
			expectedProxyName:   "node-suf",
			expectedDialerProxy: "base-suf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := map[string]any{
				"name":   "node",
				"type":   "socks5",
				"server": "127.0.0.1",
				"port":   1080,
			}
			if tt.proxyDialerProxy != "" {
				proxy["dialer-proxy"] = tt.proxyDialerProxy
			}
			payload, err := yaml.Marshal(ProxySchema{
				Proxies: []map[string]any{proxy},
			})
			if err != nil {
				t.Fatal(err)
			}

			parser, err := NewProxiesParser(
				"provider",
				nil,
				"",
				"",
				"",
				tt.providerDialerProxy,
				overrideSchema{
					AdditionalPrefix: tt.additionalPrefix,
					AdditionalSuffix: tt.additionalSuffix,
				},
				"",
			)
			if err != nil {
				t.Fatal(err)
			}

			proxies, err := parser(payload)
			if err != nil {
				t.Fatal(err)
			}
			if len(proxies) != 1 {
				t.Fatalf("got %d proxies, want 1", len(proxies))
			}

			proxyInfo := proxies[0].ProxyInfo()
			if proxies[0].Name() != tt.expectedProxyName {
				t.Fatalf("proxy name = %q, want %q", proxies[0].Name(), tt.expectedProxyName)
			}
			if proxyInfo.DialerProxy != tt.expectedDialerProxy {
				t.Fatalf("dialer-proxy = %q, want %q", proxyInfo.DialerProxy, tt.expectedDialerProxy)
			}
		})
	}
}
