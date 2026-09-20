// Package clientip resolves the real client IP behind trusted reverse proxies.
//
// Forwarding headers (X-Forwarded-For, X-Real-IP, True-Client-IP) are set by whoever sends
// the request, so they may only be believed when the TCP peer is a trusted proxy (the
// ingress controller). X-Forwarded-For is walked from the right: each trusted proxy appends
// the address it received the request from, so the right-most entry that is NOT a trusted
// proxy is the client. Anything to the left of it was supplied by the client and is ignored.
package clientip

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
)

// EnvTrustedProxies is the env var with a comma-separated list of trusted proxy CIDRs/IPs.
// Unset: DefaultTrustedProxies. Set to "none": trust no proxy (use the TCP peer only).
const EnvTrustedProxies = "TRUSTED_PROXIES"

// DefaultTrustedProxies covers loopback and private networks, where in-cluster ingress
// controllers and sidecars live. Public addresses are never trusted by default.
var DefaultTrustedProxies = []string{
	"127.0.0.0/8", "::1/128",
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
	"100.64.0.0/10", // carrier-grade NAT / some CNI pod ranges
	"fc00::/7",
}

// Resolver extracts the client IP given a set of trusted proxy networks.
type Resolver struct {
	trusted []netip.Prefix
}

// New builds a Resolver from CIDRs or bare IPs. Invalid entries are skipped.
func New(trusted []string) *Resolver {
	r := &Resolver{}
	for _, raw := range trusted {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if p, err := netip.ParsePrefix(raw); err == nil {
			r.trusted = append(r.trusted, p.Masked())
			continue
		}
		if a, err := netip.ParseAddr(raw); err == nil {
			a = a.Unmap()
			r.trusted = append(r.trusted, netip.PrefixFrom(a, a.BitLen()))
		}
	}
	return r
}

// FromEnv builds a Resolver from TRUSTED_PROXIES (see EnvTrustedProxies).
func FromEnv() *Resolver {
	v := strings.TrimSpace(os.Getenv(EnvTrustedProxies))
	switch {
	case v == "":
		return New(DefaultTrustedProxies)
	case strings.EqualFold(v, "none"):
		return New(nil)
	default:
		return New(strings.Split(v, ","))
	}
}

// Default is the process-wide resolver configured from the environment.
var Default = FromEnv()

func (r *Resolver) isTrusted(a netip.Addr) bool {
	a = a.Unmap()
	for _, p := range r.trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// ClientIP returns the client IP for r (without port).
func (r *Resolver) ClientIP(req *http.Request) string {
	peer, ok := parseHostPort(req.RemoteAddr)
	if !ok {
		return hostOnly(req.RemoteAddr)
	}
	if !r.isTrusted(peer) {
		return peer.String()
	}

	// The peer is a trusted proxy: walk X-Forwarded-For right-to-left.
	hops := forwardedFor(req.Header)
	for i := len(hops) - 1; i >= 0; i-- {
		a, ok := parseAddr(hops[i])
		if !ok {
			// Garbage at this position was injected by the client; stop here and fall back
			// to the last trustworthy hop.
			break
		}
		if !r.isTrusted(a) {
			return a.String()
		}
		peer = a
	}

	// No XFF (or only trusted hops): a trusted proxy may pass the client in X-Real-IP.
	if len(hops) == 0 {
		if a, ok := parseAddr(req.Header.Get("X-Real-IP")); ok {
			return a.String()
		}
	}
	return peer.String()
}

// Middleware rewrites req.RemoteAddr to the resolved client IP so that downstream handlers,
// loggers and rate limiters see the real client. It replaces chi's middleware.RealIP, which
// trusts forwarding headers from any peer.
func (r *Resolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ip := r.ClientIP(req); ip != "" {
			req.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		next.ServeHTTP(w, req)
	})
}

// FromRequest resolves with the Default resolver.
func FromRequest(req *http.Request) string {
	return Default.ClientIP(req)
}

func forwardedFor(h http.Header) []string {
	var hops []string
	for _, line := range h.Values("X-Forwarded-For") {
		for _, part := range strings.Split(line, ",") {
			if part = strings.TrimSpace(part); part != "" {
				hops = append(hops, part)
			}
		}
	}
	return hops
}

func parseAddr(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, false
	}
	if a, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
		return a.Unmap(), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	return netip.Addr{}, false
}

func parseHostPort(s string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	return parseAddr(s)
}

func hostOnly(s string) string {
	if host, _, err := net.SplitHostPort(s); err == nil {
		return host
	}
	return s
}
