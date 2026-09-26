package auth_test

import (
	"bytes"
	"testing"

	"github.com/Revati-Firke/gitactionflow/backend/internal/auth"
)

func TestRandomURLToken_Entropy(t *testing.T) {
	a, err := auth.RandomURLToken(32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := auth.RandomURLToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("tokens should not be identical")
	}
	if len(a) < 40 {
		t.Fatalf("token too short: %q", a)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := auth.HashToken("abc")
	h2 := auth.HashToken("abc")
	h3 := auth.HashToken("xyz")
	if !bytes.Equal(h1, h2) {
		t.Fatal("same input should hash equally")
	}
	if bytes.Equal(h1, h3) {
		t.Fatal("different input should differ")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := auth.DeriveKey("test-session-secret-at-least-32b")
	plain := []byte("gho_test_access_token_value")
	enc, err := auth.Encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(enc, plain) {
		t.Fatal("ciphertext should not contain plaintext")
	}
	out, err := auth.Decrypt(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("got %q want %q", out, plain)
	}
}
