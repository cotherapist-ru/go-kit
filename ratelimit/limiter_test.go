package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterAllow(t *testing.T) {
	l := New(2, time.Minute)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two should allow")
	}
	if l.Allow("a") {
		t.Fatal("third should deny")
	}
	if !l.Allow("b") {
		t.Fatal("other key should allow")
	}
}

func TestLimiterWindow(t *testing.T) {
	l := New(1, 20*time.Millisecond)
	if !l.Allow("k") {
		t.Fatal("first allow")
	}
	if l.Allow("k") {
		t.Fatal("within window deny")
	}
	time.Sleep(30 * time.Millisecond)
	if !l.Allow("k") {
		t.Fatal("after window allow")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if got := ClientIP(req); got != "1.2.3.4" {
		t.Fatalf("xff=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "9.9.9.9")
	if got := ClientIP(req); got != "9.9.9.9" {
		t.Fatalf("xri=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	if got := ClientIP(req); got != "10.0.0.1" {
		t.Fatalf("remote=%q", got)
	}
}
