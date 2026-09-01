package logging

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type loggerConfig struct {
	extraPaths []string
	prefixes   []string
}

// Option configures RequestLogger.
type Option func(*loggerConfig)

// WithNoisePaths appends exact paths that are logged at debug instead of info.
func WithNoisePaths(paths ...string) Option {
	return func(c *loggerConfig) {
		c.extraPaths = append(c.extraPaths, paths...)
	}
}

// WithNoisePrefixes appends path prefixes treated as noise (e.g. "/static/").
func WithNoisePrefixes(prefixes ...string) Option {
	return func(c *loggerConfig) {
		c.prefixes = append(c.prefixes, prefixes...)
	}
}

// RequestLogger logs HTTP requests with status, duration and request_id.
func RequestLogger(opts ...Option) func(http.Handler) http.Handler {
	cfg := loggerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			reqID := middleware.GetReqID(r.Context())
			reqLog := slog.Default().With(
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
			)
			ctx := WithLogger(r.Context(), reqLog)
			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []any{
				"status", status,
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			}
			if ua := r.UserAgent(); ua != "" {
				attrs = append(attrs, "user_agent", ua)
			}

			switch {
			case isNoisePath(r.URL.Path, cfg):
				reqLog.Debug("http request", attrs...)
			case status >= 500:
				reqLog.Error("http request", attrs...)
			case status >= 400:
				reqLog.Warn("http request", attrs...)
			default:
				reqLog.Info("http request", attrs...)
			}
		})
	}
}

func isNoisePath(path string, cfg loggerConfig) bool {
	switch path {
	case "/health", "/healthz", "/api/v1/health":
		return true
	}
	for _, p := range cfg.extraPaths {
		if path == p {
			return true
		}
	}
	for _, prefix := range cfg.prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
