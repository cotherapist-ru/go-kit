package servicetoken

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type options struct {
	unauthorized func(http.ResponseWriter, *http.Request)
}

// Option configures Middleware.
type Option func(*options)

// WithUnauthorized replaces the default JSON 401 writer.
func WithUnauthorized(fn func(http.ResponseWriter, *http.Request)) Option {
	return func(o *options) {
		o.unauthorized = fn
	}
}

// Middleware enforces Bearer (or raw token) auth when expected token is non-empty.
// Empty expected token disables auth (local/dev convenience).
func Middleware(expectedToken string, opts ...Option) func(http.Handler) http.Handler {
	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.unauthorized == nil {
		cfg.unauthorized = defaultUnauthorized
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(expectedToken) == "" {
				next.ServeHTTP(w, r)
				return
			}
			got := extractToken(r.Header.Get("Authorization"))
			if !secureEqual(got, expectedToken) {
				cfg.unauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func defaultUnauthorized(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "unauthorized",
		"message": "Unauthorized",
	})
}

func extractToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return header
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
