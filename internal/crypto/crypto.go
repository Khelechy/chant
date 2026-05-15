// Package crypto implements authenticated encryption for CHANT messages.
package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const nonceSize = chacha20poly1305.NonceSize

var errCiphertextTooShort = errors.New("ciphertext too short")

// Encrypt returns nonce(12) || ciphertext || tag(16).
func Encrypt(key [32]byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("chant: create cipher: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("chant: generate nonce: %w", err)
	}

	sealed := aead.Seal(nil, nonce, plaintext, nil)
	blob := make([]byte, 0, len(nonce)+len(sealed))
	blob = append(blob, nonce...)
	blob = append(blob, sealed...)
	return blob, nil
}

// Decrypt parses nonce, decrypts, and verifies the authentication tag.
func Decrypt(key [32]byte, blob []byte) ([]byte, error) {
	if len(blob) < nonceSize+chacha20poly1305.Overhead {
		return nil, fmt.Errorf("chant: blob too short: %w", errCiphertextTooShort)
	}

	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("chant: create cipher: %w", err)
	}

	nonce := blob[:nonceSize]
	ciphertext := blob[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("chant: decrypt blob: %w", err)
	}
	return plaintext, nil
}
