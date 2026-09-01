package captcha

import "context"

// Verifier checks a Yandex SmartCaptcha token.
type Verifier interface {
	Enabled() bool
	Verify(ctx context.Context, token, clientIP string) error
}

// Config holds optional SmartCaptcha keys.
type Config struct {
	ClientKey string
	ServerKey string
}

// Enabled reports whether both keys are set.
func (c Config) Enabled() bool {
	return c.ClientKey != "" && c.ServerKey != ""
}
