package ratelimit

import (
	"fmt"
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

func TestLimiterBoundsMemory(t *testing.T) {
	l := New(1, time.Minute).WithMaxKeys(10)
	for i := 0; i < 10; i++ {
		if !l.Allow(fmt.Sprintf("k%d", i)) {
			t.Fatalf("key %d should be allowed", i)
		}
	}
	if l.Allow("overflow") {
		t.Fatal("new key must be denied when table is full")
	}
	if l.Len() != 10 {
		t.Fatalf("len=%d", l.Len())
	}
}

func TestLimiterSweepsExpiredKeys(t *testing.T) {
	l := New(1, 10*time.Millisecond).WithMaxKeys(5)
	for i := 0; i < 5; i++ {
		l.Allow(fmt.Sprintf("k%d", i))
	}
	time.Sleep(20 * time.Millisecond)
	if !l.Allow("fresh") {
		t.Fatal("expired keys should be swept to make room")
	}
	if l.Len() != 1 {
		t.Fatalf("len=%d", l.Len())
	}
}

func TestClientIPIgnoresSpoofedHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "9.9.9.9")
	if got := ClientIP(req); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if got := ClientIP(req); got != "5.6.7.8" {
		t.Fatalf("got %q", got)
	}
}
