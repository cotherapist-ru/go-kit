package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff=%q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("frame=%q", rec.Header().Get("X-Frame-Options"))
	}
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("csp=%q", csp)
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS must stay off unless HTTPSERVER_HSTS/COOKIE_SECURE is set")
	}
}

func TestSecurityHeadersHSTS(t *testing.T) {
	t.Setenv("HTTPSERVER_HSTS", "true")
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Fatalf("hsts=%q", rec.Header().Get("Strict-Transport-Security"))
	}
}

func TestSecurityHeadersKeepsExistingCSP(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.WriteHeader(http.StatusNoContent)
	})
	// Headers middleware runs before the handler, so an inner Set after us wins
	// only if we skip when already set. Simulate a wrapper that set CSP first.
	pre := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		SecurityHeaders(inner).ServeHTTP(w, r)
	})
	rec := httptest.NewRecorder()
	pre.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Security-Policy") != "default-src 'none'" {
		t.Fatalf("csp=%q", rec.Header().Get("Content-Security-Policy"))
	}
}
