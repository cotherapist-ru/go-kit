package admintoken

import (
	"net/http"
	"strings"
)

type options struct {
	allowQuery        bool
	requireConfigured bool
}

// Option configures Middleware.
type Option func(*options)

// WithQueryToken also accepts ?token= in addition to Authorization: Bearer.
func WithQueryToken() Option {
	return func(o *options) {
		o.allowQuery = true
	}
}

// RequireConfigured returns 503 when the token is empty instead of 401.
func RequireConfigured() Option {
	return func(o *options) {
		o.requireConfigured = true
	}
}

// Middleware protects admin API routes with a shared token.
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
			if !authorized(r, adminToken, cfg.allowQuery) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authorized(r *http.Request, adminToken string, allowQuery bool) bool {
	if allowQuery {
		if token := r.URL.Query().Get("token"); token != "" && token == adminToken {
			return true
		}
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ") == adminToken
	}
	return false
}
