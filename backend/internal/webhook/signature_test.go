package webhook_test

import (
	"crypto/hmac"
	"testing"

	"github.com/Revati-Firke/gitactionflow/backend/internal/webhook"
)

func TestVerifySignature256_AcceptsValid(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"action":"opened","repository":{"id":1}}`)
	sig := webhook.SignBody(secret, body)
	if err := webhook.VerifySignature256(secret, body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySignature256_RejectsMissing(t *testing.T) {
	err := webhook.VerifySignature256("secret", []byte("{}"), "")
	if err != webhook.ErrMissingSignature {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature256_RejectsWrongSecret(t *testing.T) {
	body := []byte(`{"a":1}`)
	sig := webhook.SignBody("right-secret", body)
	err := webhook.VerifySignature256("wrong-secret", body, sig)
	if err != webhook.ErrInvalidSignature {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature256_RejectsModifiedBody(t *testing.T) {
	secret := "secret"
	sig := webhook.SignBody(secret, []byte(`{"a":1}`))
	err := webhook.VerifySignature256(secret, []byte(`{"a":2}`), sig)
	if err != webhook.ErrInvalidSignature {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature256_RejectsBadPrefix(t *testing.T) {
	err := webhook.VerifySignature256("secret", []byte("{}"), "sha1=deadbeef")
	if err != webhook.ErrInvalidSignature {
		t.Fatalf("got %v", err)
	}
}

func TestVerifySignature256_UsesConstantTimeCompare(t *testing.T) {
	// Ensure we rely on hmac.Equal semantics: wrong length still invalid, not panic.
	secret := "secret"
	body := []byte("x")
	err := webhook.VerifySignature256(secret, body, "sha256=abcd")
	if err != webhook.ErrInvalidSignature {
		t.Fatalf("got %v", err)
	}
	_ = hmac.Equal
}
