package builtin

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func validateFetchURL(raw string, allowLocal bool) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http and https URLs are allowed")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url host is required")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("url host is required")
	}
	if !allowLocal {
		if err := rejectLocalHost(host); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func rejectLocalHost(host string) error {
	lower := strings.ToLower(strings.TrimSpace(host))
	if lower == "localhost" {
		return fmt.Errorf("fetch to localhost is blocked (set tools.allow_local_fetch: true to allow)")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if blockedFetchIP(ip) {
			return fmt.Errorf("fetch to private or local address %s is blocked (set tools.allow_local_fetch: true to allow)", ip.String())
		}
	}
	return nil
}

func blockedFetchIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	switch {
	case ip4[0] == 10:
		return true
	case ip4[0] == 127:
		return true
	case ip4[0] == 169 && ip4[1] == 254:
		return true
	case ip4[0] == 192 && ip4[1] == 168:
		return true
	default:
		return false
	}
}
