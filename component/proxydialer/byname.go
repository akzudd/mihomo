package proxydialer

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
)

type byNameProxyDialer struct {
	proxyName string
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
	// First, try to find proxy in tunnel.Proxies()
	proxies := tunnel.Proxies()
	if proxy, ok := proxies[d.proxyName]; ok {
		return proxy, nil
	}

	// If not found, try to find proxy in all providers
	providers := tunnel.Providers()
	for _, provider := range providers {
		for _, proxy := range provider.Proxies() {
			if proxy.Name() == d.proxyName {
				return proxy, nil
			}
		}
	}

	return nil, fmt.Errorf("proxyName[%s] not found", d.proxyName)
}

func NewByName(proxyName string) C.Dialer {
	return byNameProxyDialer{proxyName: proxyName}
}
