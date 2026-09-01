package captcha

import (
	"fmt"
	"os"
)

// Load reads YANDEX_CAPTCHA_* keys. Both empty is OK; only one set is an error.
func Load() (Config, error) {
	clientKey := os.Getenv("YANDEX_CAPTCHA_CLIENT_KEY")
	serverKey := os.Getenv("YANDEX_CAPTCHA_SERVER_KEY")

	if clientKey == "" && serverKey == "" {
		return Config{}, nil
	}
	if clientKey == "" || serverKey == "" {
		return Config{}, fmt.Errorf("YANDEX_CAPTCHA_CLIENT_KEY and YANDEX_CAPTCHA_SERVER_KEY must both be set")
	}

	return Config{
		ClientKey: clientKey,
		ServerKey: serverKey,
	}, nil
}

// NewFromConfig returns a nop verifier when captcha is disabled.
func NewFromConfig(cfg Config) Verifier {
	if !cfg.Enabled() {
		return NewNopVerifier()
	}
	return NewYandexVerifier(cfg.ServerKey)
}
