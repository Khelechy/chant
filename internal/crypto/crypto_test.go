package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := [32]byte{1, 2, 3, 4, 5}
	plaintext := []byte("hello chant")

	blob, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	got, err := Decrypt(key, blob)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Fatalf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key := [32]byte{1, 2, 3, 4, 5}
	wrongKey := [32]byte{5, 4, 3, 2, 1}

	blob, err := Encrypt(key, []byte("hello chant"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if _, err := Decrypt(wrongKey, blob); err == nil {
		t.Fatal("Decrypt() error = nil, want authentication failure")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	key := [32]byte{1, 2, 3, 4, 5}

	blob, err := Encrypt(key, []byte("hello chant"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	blob[len(blob)-1] ^= 0xFF

	if _, err := Decrypt(key, blob); err == nil {
		t.Fatal("Decrypt() error = nil, want authentication failure")
	}
}
