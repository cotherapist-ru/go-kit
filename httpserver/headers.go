package httpserver

import (
	"net/http"
	"os"
	"strings"
)

// defaultCSP is a baseline policy for public Go services. Inline style/script are
// allowed because existing promo/feedback/public-testing pages use them. Object
// plugins and framing are forbidden. A service can send a stricter CSP afterwards;
// browsers AND multiple policies together.
const defaultCSP = "default-src 'self'; script-src 'self' 'unsafe-inline' https://smartcaptcha.yandexcloud.net https://mc.yandex.ru; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob: https:; connect-src 'self' https://smartcaptcha.yandexcloud.net https://mc.yandex.ru; object-src 'none'; base-uri 'none'; frame-ancestors 'none'"

// SecurityHeaders sets nosniff, anti-clickjacking, Referrer-Policy, a baseline CSP
// and (when HTTPSERVER_HSTS or COOKIE_SECURE is set) HSTS.
func SecurityHeaders(next http.Handler) http.Handler {
	hsts := envTruthy("HTTPSERVER_HSTS") || envTruthy("COOKIE_SECURE")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		if h.Get("Content-Security-Policy") == "" {
			h.Set("Content-Security-Policy", defaultCSP)
		}
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		if hsts {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
