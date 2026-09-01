package logging

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey struct{}

// WithLogger stores a request-scoped logger in the context.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	if log == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext returns the logger from context, a request_id-enriched default, or slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	if rid := middleware.GetReqID(ctx); rid != "" {
		return slog.Default().With("request_id", rid)
	}
	return slog.Default()
}

// FromRequest returns a logger enriched with chi request_id.
func FromRequest(r *http.Request) *slog.Logger {
	if r == nil {
		return slog.Default()
	}
	return FromContext(r.Context())
}
