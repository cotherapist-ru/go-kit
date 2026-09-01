package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cotherapist-ru/go-kit/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const defaultTimeout = 60 * time.Second

type routerOptions struct {
	timeout    *time.Duration
	logger     func(http.Handler) http.Handler
	skipLogger bool
}

// RouterOption configures NewRouter.
type RouterOption func(*routerOptions)

// WithTimeout sets chi Timeout middleware. Zero duration skips Timeout.
func WithTimeout(d time.Duration) RouterOption {
	return func(o *routerOptions) {
		o.timeout = &d
	}
}

// WithoutTimeout omits chi Timeout middleware.
func WithoutTimeout() RouterOption {
	zero := time.Duration(0)
	return func(o *routerOptions) {
		o.timeout = &zero
	}
}

// WithLogger replaces the default request logger middleware.
func WithLogger(mw func(http.Handler) http.Handler) RouterOption {
	return func(o *routerOptions) {
		o.logger = mw
	}
}

// WithoutLogger skips request logging middleware.
func WithoutLogger() RouterOption {
	return func(o *routerOptions) {
		o.skipLogger = true
	}
}

// NewRouter returns a chi router with RequestID, RealIP, request logger, Recoverer and Timeout.
func NewRouter(opts ...RouterOption) *chi.Mux {
	cfg := routerOptions{}
	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if !cfg.skipLogger {
		logger := cfg.logger
		if logger == nil {
			logger = logging.RequestLogger()
		}
		r.Use(logger)
	}
	r.Use(middleware.Recoverer)

	timeout := defaultTimeout
	if cfg.timeout != nil {
		timeout = *cfg.timeout
	}
	if timeout > 0 {
		r.Use(middleware.Timeout(timeout))
	}
	return r
}

// Options configures ListenAndServe / Run.
type Options struct {
	Addr    string
	Handler http.Handler
}

// Run starts the server and blocks until SIGINT/SIGTERM.
func Run(opts Options) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := Serve(ctx, opts); err != nil && err != http.ErrServerClosed && err != context.Canceled {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}

// Serve listens until ctx is cancelled, then shuts down with a 10s timeout.
func Serve(ctx context.Context, opts Options) error {
	server := &http.Server{
		Addr:              opts.Addr,
		Handler:           opts.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", opts.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "error", err)
			return err
		}
		return <-errCh
	}
}
