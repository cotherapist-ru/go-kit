package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/cotherapist-ru/go-kit/clientip"
)

// DefaultMaxKeys bounds the number of tracked keys so that the limiter cannot be used to
// exhaust memory. When the table is full and a sweep of expired keys frees nothing, new keys
// are denied (fail closed) until old entries expire.
const DefaultMaxKeys = 100_000

// sweepEvery triggers a sweep of expired keys every N calls to Allow.
const sweepEvery = 1024

type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	maxKeys int
	calls   int
	entries map[string][]time.Time
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		limit:   limit,
		window:  window,
		maxKeys: DefaultMaxKeys,
		entries: make(map[string][]time.Time),
	}
}

// WithMaxKeys overrides DefaultMaxKeys.
func (l *Limiter) WithMaxKeys(n int) *Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n > 0 {
		l.maxKeys = n
	}
	return l
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	if l.calls%sweepEvery == 0 {
		l.sweep(cutoff)
	}

	times, known := l.entries[key]
	if !known && len(l.entries) >= l.maxKeys {
		l.sweep(cutoff)
		if len(l.entries) >= l.maxKeys {
			return false
		}
	}

	filtered := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= l.limit {
		l.entries[key] = filtered
		return false
	}

	filtered = append(filtered, now)
	l.entries[key] = filtered
	return true
}

// Len reports the number of tracked keys (for tests and metrics).
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// sweep drops keys whose most recent hit is outside the window. Caller holds l.mu.
func (l *Limiter) sweep(cutoff time.Time) {
	for key, times := range l.entries {
		if len(times) == 0 || !times[len(times)-1].After(cutoff) {
			delete(l.entries, key)
		}
	}
}

// ClientIP returns the client IP for rate limiting.
//
// Forwarding headers are honoured only when the TCP peer is a trusted proxy (see package
// clientip, TRUSTED_PROXIES). Previously the first X-Forwarded-For value was used, which the
// client controls, so any limit could be bypassed by rotating that header.
func ClientIP(r *http.Request) string {
	return clientip.FromRequest(r)
}
