package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func req(remote string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remote
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestClientIP(t *testing.T) {
	res := New(DefaultTrustedProxies)
	cases := []struct {
		name    string
		remote  string
		headers map[string]string
		want    string
	}{
		{"direct client ignores spoofed XFF", "203.0.113.7:5555", map[string]string{"X-Forwarded-For": "1.1.1.1"}, "203.0.113.7"},
		{"direct client ignores spoofed X-Real-IP", "203.0.113.7:5555", map[string]string{"X-Real-IP": "1.1.1.1"}, "203.0.113.7"},
		{"behind ingress uses right-most untrusted", "10.0.0.5:80", map[string]string{"X-Forwarded-For": "203.0.113.7"}, "203.0.113.7"},
		{"client-prepended values are ignored", "10.0.0.5:80", map[string]string{"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 203.0.113.7"}, "203.0.113.7"},
		{"multiple trusted hops are skipped", "10.0.0.5:80", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7, 10.1.2.3"}, "203.0.113.7"},
		{"garbage injected before trusted hop stops the walk", "10.0.0.5:80", map[string]string{"X-Forwarded-For": "203.0.113.7, not-an-ip, 10.1.2.3"}, "10.1.2.3"},
		{"trusted peer without XFF uses X-Real-IP", "10.0.0.5:80", map[string]string{"X-Real-IP": "203.0.113.9"}, "203.0.113.9"},
		{"trusted peer without headers", "10.0.0.5:80", nil, "10.0.0.5"},
		{"ipv6 peer", "[2001:db8::1]:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}, "2001:db8::1"},
		{"ipv4-mapped ipv6 trusted peer", "[::ffff:10.0.0.5]:80", map[string]string{"X-Forwarded-For": "203.0.113.7"}, "203.0.113.7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := res.ClientIP(req(tc.remote, tc.headers)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNoTrustedProxies(t *testing.T) {
	res := New(nil)
	if got := res.ClientIP(req("10.0.0.5:80", map[string]string{"X-Forwarded-For": "203.0.113.7"})); got != "10.0.0.5" {
		t.Fatalf("got %q", got)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv(EnvTrustedProxies, "198.51.100.0/24, 192.0.2.1")
	res := FromEnv()
	if got := res.ClientIP(req("192.0.2.1:80", map[string]string{"X-Forwarded-For": "203.0.113.7"})); got != "203.0.113.7" {
		t.Fatalf("bare IP not trusted: %q", got)
	}
	if got := res.ClientIP(req("10.0.0.5:80", map[string]string{"X-Forwarded-For": "203.0.113.7"})); got != "10.0.0.5" {
		t.Fatalf("private range must not be trusted when list is explicit: %q", got)
	}

	t.Setenv(EnvTrustedProxies, "none")
	if got := FromEnv().ClientIP(req("127.0.0.1:80", map[string]string{"X-Forwarded-For": "203.0.113.7"})); got != "127.0.0.1" {
		t.Fatalf("none: %q", got)
	}
}

func TestMiddlewareRewritesRemoteAddr(t *testing.T) {
	res := New(DefaultTrustedProxies)
	var seen string
	h := res.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.RemoteAddr }))
	h.ServeHTTP(httptest.NewRecorder(), req("10.0.0.5:80", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7"}))
	if seen != "203.0.113.7:0" {
		t.Fatalf("RemoteAddr=%q", seen)
	}
}
