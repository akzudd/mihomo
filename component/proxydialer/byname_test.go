package proxydialer

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/metacubex/mihomo/common/utils"
	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
)

type testTunnel struct {
	proxies   map[string]C.Proxy
	providers map[string]P.ProxyProvider
}

func (t testTunnel) HandleTCPConn(net.Conn, *C.Metadata)      {}
func (t testTunnel) HandleUDPPacket(C.UDPPacket, *C.Metadata) {}
func (t testTunnel) NatTable() C.NatTable                     { return nil }
func (t testTunnel) Proxies() map[string]C.Proxy {
	return t.proxies
}
func (t testTunnel) Providers() map[string]P.ProxyProvider {
	return t.providers
}

type testProxyOnlyTunnel struct {
	proxies map[string]C.Proxy
}

func (t testProxyOnlyTunnel) HandleTCPConn(net.Conn, *C.Metadata)      {}
func (t testProxyOnlyTunnel) HandleUDPPacket(C.UDPPacket, *C.Metadata) {}
func (t testProxyOnlyTunnel) NatTable() C.NatTable                     { return nil }
func (t testProxyOnlyTunnel) Proxies() map[string]C.Proxy {
	return t.proxies
}

type testProxyProvider struct {
	name    string
	proxies []C.Proxy
}

func (p testProxyProvider) Name() string               { return p.name }
func (p testProxyProvider) VehicleType() P.VehicleType { return P.Inline }
func (p testProxyProvider) Type() P.ProviderType       { return P.Proxy }
func (p testProxyProvider) Initial() error             { return nil }
func (p testProxyProvider) Update() error              { return nil }
func (p testProxyProvider) Proxies() []C.Proxy         { return p.proxies }
func (p testProxyProvider) Count() int                 { return len(p.proxies) }
func (p testProxyProvider) Touch()                     {}
func (p testProxyProvider) HealthCheck()               {}
func (p testProxyProvider) Version() uint32            { return 0 }
func (p testProxyProvider) RegisterHealthCheckTask(string, utils.IntRanges[uint16], string, uint) {
}
func (p testProxyProvider) HealthCheckURL() string { return "" }

type testProxy struct {
	name string
}

func (p testProxy) Adapter() C.ProxyAdapter     { return p }
func (p testProxy) AliveForTestUrl(string) bool { return true }
func (p testProxy) DelayHistory() []C.DelayHistory {
	return []C.DelayHistory{}
}
func (p testProxy) ExtraDelayHistories() map[string]C.ProxyState {
	return map[string]C.ProxyState{}
}
func (p testProxy) LastDelayForTestUrl(string) uint16 { return 0 }
func (p testProxy) URLTest(context.Context, string, utils.IntRanges[uint16]) (uint16, error) {
	return 0, nil
}
func (p testProxy) Name() string        { return p.name }
func (p testProxy) Type() C.AdapterType { return C.Direct }
func (p testProxy) Addr() string        { return "" }
func (p testProxy) SupportUDP() bool    { return true }
func (p testProxy) ProxyInfo() C.ProxyInfo {
	return C.ProxyInfo{}
}
func (p testProxy) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"name": p.name})
}
func (p testProxy) DialContext(context.Context, *C.Metadata) (C.Conn, error) {
	return nil, C.ErrNotSupport
}
func (p testProxy) ListenPacketContext(context.Context, *C.Metadata) (C.PacketConn, error) {
	return nil, C.ErrNotSupport
}
func (p testProxy) SupportUOT() bool { return false }
func (p testProxy) IsL3Protocol(*C.Metadata) bool {
	return false
}
func (p testProxy) Unwrap(*C.Metadata, bool) C.Proxy { return nil }
func (p testProxy) Close() error                     { return nil }

func TestByNameProxyDialerFindProxy(t *testing.T) {
	globalProxy := testProxy{name: "target"}
	providerProxy := testProxy{name: "target"}

	tests := []struct {
		name      string
		tunnel    testTunnel
		wantProxy C.Proxy
		wantErr   string
	}{
		{
			name: "global proxy takes precedence",
			tunnel: testTunnel{
				proxies: map[string]C.Proxy{
					"target": globalProxy,
				},
				providers: map[string]P.ProxyProvider{
					"provider": testProxyProvider{
						name:    "provider",
						proxies: []C.Proxy{providerProxy},
					},
				},
			},
			wantProxy: globalProxy,
		},
		{
			name: "provider proxy is used when global proxy is missing",
			tunnel: testTunnel{
				proxies: map[string]C.Proxy{},
				providers: map[string]P.ProxyProvider{
					"provider": testProxyProvider{
						name:    "provider",
						proxies: []C.Proxy{providerProxy},
					},
				},
			},
			wantProxy: providerProxy,
		},
		{
			name: "missing proxy returns not found error",
			tunnel: testTunnel{
				proxies:   map[string]C.Proxy{},
				providers: map[string]P.ProxyProvider{},
			},
			wantErr: "proxyName[target] not found",
		},
		{
			name: "multiple provider proxies return ambiguous error",
			tunnel: testTunnel{
				proxies: map[string]C.Proxy{},
				providers: map[string]P.ProxyProvider{
					"provider-a": testProxyProvider{
						name:    "provider-a",
						proxies: []C.Proxy{testProxy{name: "target"}},
					},
					"provider-b": testProxyProvider{
						name:    "provider-b",
						proxies: []C.Proxy{testProxy{name: "target"}},
					},
				},
			},
			wantErr: "proxyName[target] is ambiguous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialer := byNameProxyDialer{
				proxyName: "target",
				tunnel:    tt.tunnel,
			}

			got, err := dialer.findProxy()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("findProxy() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("findProxy() error = %q, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("findProxy() error = %v", err)
			}
			if got != tt.wantProxy {
				t.Fatalf("findProxy() = %v, want %v", got, tt.wantProxy)
			}
		})
	}
}

func TestByNameProxyDialerFindProxyUsesGlobalProxyWithoutProviders(t *testing.T) {
	globalProxy := testProxy{name: "target"}
	dialer := byNameProxyDialer{
		proxyName: "target",
		tunnel: testProxyOnlyTunnel{
			proxies: map[string]C.Proxy{
				"target": globalProxy,
			},
		},
	}

	got, err := dialer.findProxy()
	if err != nil {
		t.Fatalf("findProxy() error = %v", err)
	}
	if got != globalProxy {
		t.Fatalf("findProxy() = %v, want %v", got, globalProxy)
	}
}
