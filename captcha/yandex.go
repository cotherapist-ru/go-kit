package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultValidateURL = "https://smartcaptcha.cloud.yandex.ru/validate"

var ErrInvalidCaptcha = errors.New("invalid captcha")
var ErrMissingCaptcha = errors.New("missing captcha token")

// YandexVerifier validates tokens against Yandex SmartCaptcha.
type YandexVerifier struct {
	serverKey   string
	client      *http.Client
	validateURL string
}

// NewYandexVerifier builds a verifier with a 5s HTTP timeout.
func NewYandexVerifier(serverKey string) *YandexVerifier {
	return &YandexVerifier{
		serverKey: serverKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		validateURL: defaultValidateURL,
	}
}

// SetValidateURL overrides the validation endpoint (tests).
func (y *YandexVerifier) SetValidateURL(u string) {
	y.validateURL = u
}

// Enabled reports whether a server key is configured.
func (y *YandexVerifier) Enabled() bool {
	return y.serverKey != ""
}

// Verify posts the token to Yandex SmartCaptcha.
func (y *YandexVerifier) Verify(ctx context.Context, token, clientIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrMissingCaptcha
	}

	data := url.Values{}
	data.Set("secret", y.serverKey)
	data.Set("token", token)
	if clientIP != "" {
		data.Set("ip", clientIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.validateURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("build captcha request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := y.client.Do(req)
	if err != nil {
		return fmt.Errorf("captcha request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read captcha response: %w", err)
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode captcha response: %w", err)
	}

	if result.Status == "ok" {
		return nil
	}
	return ErrInvalidCaptcha
}

var _ Verifier = (*YandexVerifier)(nil)
