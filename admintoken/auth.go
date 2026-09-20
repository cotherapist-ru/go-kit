package admintoken

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type options struct {
	requireConfigured bool
}

// Option configures Middleware.
type Option func(*options)

// WithQueryToken used to accept ?token= in addition to Authorization: Bearer.
//
// Deprecated: tokens in URLs leak through access logs, browser history and the Referer
// header, so query tokens are no longer accepted. This option is a no-op kept for source
// compatibility; send the token in the Authorization header.
func WithQueryToken() Option {
	return func(*options) {}
}

// RequireConfigured returns 503 when the token is empty instead of 401.
func RequireConfigured() Option {
	return func(o *options) {
		o.requireConfigured = true
	}
}

// Middleware protects admin API routes with a shared token sent as
// "Authorization: Bearer <token>". An empty configured token never authorizes anything.
func Middleware(adminToken string, opts ...Option) func(http.Handler) http.Handler {
	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.requireConfigured && adminToken == "" {
				http.Error(w, "admin api disabled", http.StatusServiceUnavailable)
				return
			}
			if !Authorized(r, adminToken) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Authorized reports whether r carries "Authorization: Bearer <adminToken>".
// The comparison is constant-time; an empty adminToken is never authorized.
func Authorized(r *http.Request, adminToken string) bool {
	if adminToken == "" {
		return false
	}
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return false
	}
	got := strings.TrimSpace(auth[len(prefix):])
	return subtle.ConstantTimeCompare([]byte(got), []byte(adminToken)) == 1
}
