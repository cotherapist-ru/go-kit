package captcha

import "context"

// NopVerifier always succeeds and reports disabled.
type NopVerifier struct{}

// NewNopVerifier returns a no-op verifier.
func NewNopVerifier() *NopVerifier {
	return &NopVerifier{}
}

// Enabled always returns false.
func (n *NopVerifier) Enabled() bool {
	return false
}

// Verify always succeeds.
func (n *NopVerifier) Verify(context.Context, string, string) error {
	return nil
}

var _ Verifier = (*NopVerifier)(nil)
