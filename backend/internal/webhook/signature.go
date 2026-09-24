package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const signaturePrefix = "sha256="

// VerifySignature256 validates X-Hub-Signature-256 using HMAC-SHA256 over the raw body.
// Comparison is constant-time via hmac.Equal.
func VerifySignature256(secret string, body []byte, header string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("webhook secret not configured")
	}
	header = strings.TrimSpace(header)
	if header == "" {
		return ErrMissingSignature
	}
	if !strings.HasPrefix(header, signaturePrefix) {
		return ErrInvalidSignature
	}
	gotHex := strings.TrimPrefix(header, signaturePrefix)
	got, err := hex.DecodeString(gotHex)
	if err != nil {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)

	if !hmac.Equal(expected, got) {
		return ErrInvalidSignature
	}
	return nil
}

// SignBody is a test helper that builds an X-Hub-Signature-256 value.
func SignBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}
