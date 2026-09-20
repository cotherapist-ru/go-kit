package admintoken

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerOK(t *testing.T) {
	h := Middleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestBearerRejected(t *testing.T) {
	h := Middleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestQueryTokenIgnoredEvenWithDeprecatedOption(t *testing.T) {
	h := Middleware("secret", WithQueryToken())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/?token=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestEmptyTokenNeverAuthorizes(t *testing.T) {
	h := Middleware("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, header := range []string{"", "Bearer", "Bearer ", "Bearer  "} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: code=%d", header, rec.Code)
		}
	}
}

func TestBearerPrefixCaseInsensitive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer secret")
	if !Authorized(req, "secret") {
		t.Fatal("expected authorized")
	}
}

func TestQueryTokenDisabled(t *testing.T) {
	h := Middleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/?token=secret", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestRequireConfiguredEmpty(t *testing.T) {
	h := Middleware("", RequireConfigured())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d", rec.Code)
	}
}
