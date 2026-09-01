package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"info":    slog.LevelInfo,
		"unknown": slog.LevelInfo,
	}

	for raw, want := range cases {
		if got := parseLevel(raw); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestFromContextFallback(t *testing.T) {
	if FromContext(context.Background()) == nil {
		t.Fatal("expected non-nil default logger")
	}
	if FromContext(nil) == nil {
		t.Fatal("expected non-nil default logger for nil context")
	}
}

func TestFromRequestIncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	base := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(base)
	t.Cleanup(func() { slog.SetDefault(prev) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "req-123")
	req = req.WithContext(ctx)

	FromRequest(req).Info("hello")
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("json: %v body=%s", err, buf.String())
	}
	if entry["request_id"] != "req-123" {
		t.Fatalf("entry=%v", entry)
	}
	if entry["msg"] != "hello" {
		t.Fatalf("msg=%v", entry["msg"])
	}
}

func TestWithLoggerRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil)).With("scope", "test")
	ctx := WithLogger(context.Background(), log)
	FromContext(ctx).Info("ping")
	if !bytes.Contains(buf.Bytes(), []byte(`"scope":"test"`)) {
		t.Fatalf("missing scope: %s", buf.String())
	}
}

func TestRequestLoggerEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	handler := middleware.RequestID(RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instruments/mmpi_2/attempts/submit", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected log line")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("json.Unmarshal: %v, line=%q", err, line)
	}

	if entry["msg"] != "http request" {
		t.Fatalf("msg = %v", entry["msg"])
	}
	if entry["method"] != "POST" {
		t.Fatalf("method = %v", entry["method"])
	}
	if entry["path"] != "/api/v1/instruments/mmpi_2/attempts/submit" {
		t.Fatalf("path = %v", entry["path"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("status = %v", entry["status"])
	}
	if entry["request_id"] == nil || entry["request_id"] == "" {
		t.Fatalf("expected request_id, got %v", entry["request_id"])
	}
}

func TestRequestLoggerSkipsHealthAtInfo(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	handler := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/healthz", "/api/v1/health"} {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if buf.Len() != 0 {
			t.Fatalf("expected no info log for %s, got %q", path, buf.String())
		}
	}
}

func TestRequestLoggerExtraNoisePaths(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	handler := RequestLogger(WithNoisePaths("/styles.css"), WithNoisePrefixes("/static/"))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/styles.css", "/static/app.js"} {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if buf.Len() != 0 {
			t.Fatalf("expected no info log for %s, got %q", path, buf.String())
		}
	}
}

func TestRequestLoggerWarnsOnClientError(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	handler := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, `"level":"WARN"`) && !strings.Contains(out, `"level": "WARN"`) {
		t.Fatalf("expected warn log, got %q", out)
	}
}

func TestSetupJSONDefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "")

	log := Setup("go-kit-test")
	if log == nil {
		t.Fatal("expected logger")
	}
}
