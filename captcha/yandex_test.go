package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestYandexVerifier_VerifyOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		_ = r.ParseForm()
		if r.Form.Get("secret") != "server-secret" {
			t.Fatalf("secret = %q", r.Form.Get("secret"))
		}
		if r.Form.Get("token") != "good-token" {
			t.Fatalf("token = %q", r.Form.Get("token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	v := NewYandexVerifier("server-secret")
	v.SetValidateURL(server.URL)

	if err := v.Verify(context.Background(), "good-token", "127.0.0.1"); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestYandexVerifier_VerifyFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"failed","message":"Invalid or expired Token."}`))
	}))
	defer server.Close()

	v := NewYandexVerifier("server-secret")
	v.SetValidateURL(server.URL)

	err := v.Verify(context.Background(), "bad-token", "")
	if err != ErrInvalidCaptcha {
		t.Fatalf("err = %v", err)
	}
}

func TestYandexVerifier_MissingToken(t *testing.T) {
	v := NewYandexVerifier("server-secret")
	if err := v.Verify(context.Background(), "", ""); err != ErrMissingCaptcha {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadRequiresBothKeys(t *testing.T) {
	t.Setenv("YANDEX_CAPTCHA_CLIENT_KEY", "client")
	t.Setenv("YANDEX_CAPTCHA_SERVER_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when only client key is set")
	}
}

func TestLoadBothEmpty(t *testing.T) {
	t.Setenv("YANDEX_CAPTCHA_CLIENT_KEY", "")
	t.Setenv("YANDEX_CAPTCHA_SERVER_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled() {
		t.Fatal("expected disabled")
	}
}

func TestNewFromConfigNop(t *testing.T) {
	v := NewFromConfig(Config{})
	if v.Enabled() {
		t.Fatal("expected nop")
	}
}
