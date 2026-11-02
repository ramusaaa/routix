package routix

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func (r *Router) SetTrustedProxies(proxies []string) error {
	networks := make([]*net.IPNet, 0, len(proxies))
	for _, value := range proxies {
		if ip := net.ParseIP(value); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy %q: %w", value, err)
		}
		networks = append(networks, network)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trustedProxies = networks
	r.dispatch.Store(nil)
	return nil
}

func remoteIP(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}

func trustsProxy(address string, proxies []*net.IPNet) bool {
	if len(proxies) == 0 {
		return false
	}
	ip := net.ParseIP(address)
	for _, network := range proxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func requestClientIP(req *http.Request, proxies []*net.IPNet) string {
	peer := remoteIP(req.RemoteAddr)
	if !trustsProxy(peer, proxies) {
		return peer
	}
	if values := req.Header.Values("X-Forwarded-For"); strings.TrimSpace(strings.Join(values, ",")) != "" {
		chain := strings.Split(strings.Join(values, ","), ",")
		current := peer
		for i := len(chain) - 1; i >= 0; i-- {
			if !trustsProxy(current, proxies) {
				return current
			}
			value := strings.TrimSpace(chain[i])
			if net.ParseIP(value) == nil {
				return peer
			}
			current = value
		}
		return current
	}
	if value := strings.TrimSpace(req.Header.Get("X-Real-IP")); net.ParseIP(value) != nil {
		return value
	}
	return peer
}

func requestIsSecure(req *http.Request, proxies []*net.IPNet) bool {
	if req.TLS != nil {
		return true
	}
	if len(proxies) == 0 {
		return false
	}
	values := req.Header.Values("X-Forwarded-Proto")
	return trustsProxy(remoteIP(req.RemoteAddr), proxies) && len(values) == 1 && values[0] == "https"
}

func (c *Context) IsSecure() bool    { return c.secure || c.Request.TLS != nil }
func (c *Context) RoutePath() string { return c.routePath }
