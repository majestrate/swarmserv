package network

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

// NetContext is used for all network related tasks
type NetContext struct {
	Resolver *net.Resolver
	Dialer   *net.Dialer
}

// LookupRemoteAddress gets the loki address of a remote host by ip addr
func (net *NetContext) LookupRemoteAddress(addr string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	names, err := net.Resolver.LookupAddr(ctx, addr)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", errors.New("cannot determine remote address")
	}
	return strings.TrimSuffix(names[0], "."), nil
}

// LookupOurAddress looks up what our loki address is
func (net *NetContext) LookupOurAddress() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	addrs, err := net.Resolver.LookupHost(ctx, "localhost.loki")
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", errors.New("cannot lookup our local address")
	}
	names, err := net.Resolver.LookupAddr(ctx, addrs[0])
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", errors.New("cannot determine our local address")
	}
	return strings.TrimSuffix(names[0], "."), nil
}

// CreateNetContext creates a network context for lookups and dialing out
func CreateNetContext(dnshost, dnsport string) *NetContext {
	dnsaddr := net.JoinHostPort(dnshost, dnsport)
	resolver := &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "udp", dnsaddr)
		},
	}
	return &NetContext{
		Resolver: resolver,
		Dialer: &net.Dialer{
			Resolver: resolver,
		},
	}
}
