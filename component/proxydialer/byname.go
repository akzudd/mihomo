package proxydialer

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
)

type Tunnel interface {
	C.Tunnel
	Proxies() map[string]C.Proxy
}

type providerTunnel interface {
	Tunnel
	Providers() map[string]P.ProxyProvider
}

type byNameProxyDialer struct {
	proxyName string
	tunnel    C.Tunnel
}

func (d byNameProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	proxy, err := d.findProxy()
	if err != nil {
		return nil, err
	}
	return New(proxy, true).DialContext(ctx, network, address)
}

func (d byNameProxyDialer) ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error) {
	proxy, err := d.findProxy()
	if err != nil {
		return nil, err
	}
	return New(proxy, true).ListenPacket(ctx, network, address, rAddrPort)
}

func (d byNameProxyDialer) findProxy() (C.Proxy, error) {
	tunnel, _ := d.tunnel.(Tunnel)
	if tunnel == nil {
		return nil, fmt.Errorf("tunnel is invalid, must be proxydialer.Tunnel, but got: %T", d.tunnel)
	}

	proxies := tunnel.Proxies()
	if proxy, ok := proxies[d.proxyName]; ok {
		return proxy, nil
	}

	providerTunnel, _ := d.tunnel.(providerTunnel)
	if providerTunnel == nil {
		return nil, fmt.Errorf("proxyName[%s] not found", d.proxyName)
	}

	var providerProxy C.Proxy
	providerMatches := 0
	for _, provider := range providerTunnel.Providers() {
		for _, proxy := range provider.Proxies() {
			if proxy.Name() != d.proxyName {
				continue
			}
			providerProxy = proxy
			providerMatches++
		}
	}

	switch providerMatches {
	case 0:
		return nil, fmt.Errorf("proxyName[%s] not found", d.proxyName)
	case 1:
		return providerProxy, nil
	default:
		return nil, fmt.Errorf("proxyName[%s] is ambiguous across proxy providers", d.proxyName)
	}
}

func NewByName(proxyName string, tunnel C.Tunnel) C.Dialer {
	return byNameProxyDialer{proxyName: proxyName, tunnel: tunnel}
}
