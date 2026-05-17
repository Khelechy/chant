// Package chant exposes the public CHANT encoding and decoding API.
package chant

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	chantcrypto "github.com/khelechy/chant/internal/crypto"
	"github.com/khelechy/chant/internal/fec"
	"github.com/khelechy/chant/internal/frame"
	"github.com/khelechy/chant/internal/modem"
)

// EncodeMessage runs encrypt -> FEC encode -> frame -> modulate.
func EncodeMessage(key [32]byte, plaintext []byte) ([]float32, error) {
	blob, err := chantcrypto.Encrypt(key, plaintext)
	if err != nil {
		return nil, fmt.Errorf("chant: encrypt message: %w", err)
	}

	encoded, originalLen, err := fec.FECEncode(blob)
	if err != nil {
		return nil, fmt.Errorf("chant: fec encode message: %w", err)
	}
	if originalLen > 0xFFFF {
		return nil, fmt.Errorf("chant: encrypted message too large: %w", ErrInvalidLength)
	}

	framed := frame.Frame(encoded, uint16(originalLen))
	mod := modem.NewModulator(modem.DefaultSampleRate)
	return mod.ModulatePacket(framed), nil
}

// DecodeMessage runs demodulate -> unframe -> FEC decode -> decrypt.
func DecodeMessage(key [32]byte, samples []float32) ([]byte, error) {
	return DecodeMessageWithSampleRate(key, samples, modem.DefaultSampleRate)
}

// ExtractEncryptedMessage runs demodulate -> unframe -> FEC decode and returns
// the encrypted blob carried by CHANT frames: nonce(12) || ciphertext || tag(16).
func ExtractEncryptedMessage(samples []float32) ([]byte, error) {
	return ExtractEncryptedMessageWithSampleRate(samples, modem.DefaultSampleRate)
}

// DecodeMessageWithSampleRate runs demodulate -> unframe -> FEC decode -> decrypt
// using the provided sample rate for demodulation.
func DecodeMessageWithSampleRate(key [32]byte, samples []float32, sampleRate int) ([]byte, error) {
	blob, err := ExtractEncryptedMessageWithSampleRate(samples, sampleRate)
	if err != nil {
		return nil, err
	}

	plaintext, err := DecryptMessageBlob(key, blob)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// ExtractEncryptedMessageWithSampleRate runs demodulate -> unframe -> FEC decode
// using the provided sample rate and returns the encrypted CHANT blob.
func ExtractEncryptedMessageWithSampleRate(samples []float32, sampleRate int) ([]byte, error) {
	demod := modem.NewDemodulator(sampleRate)
	framed, err := demod.Demodulate(samples)
	if err != nil {
		return nil, fmt.Errorf("chant: demodulate message: %w", err)
	}

	payload, originalLen, err := frame.Unframe(framed)
	if err != nil {
		return nil, fmt.Errorf("chant: unframe message: %w", err)
	}

	blob, err := fec.FECDecode(payload, int(originalLen))
	if err != nil {
		return nil, fmt.Errorf("chant: fec decode message: %w", err)
	}
	return blob, nil
}

// DecryptMessageBlob decrypts a CHANT encrypted blob of the form
// nonce(12) || ciphertext || tag(16).
func DecryptMessageBlob(key [32]byte, blob []byte) ([]byte, error) {
	plaintext, err := chantcrypto.Decrypt(key, blob)
	if err != nil {
		return nil, fmt.Errorf("chant: decrypt message: %w", err)
	}
	return plaintext, nil
}

// KeyFromHex parses a 64-character hex string into a 32-byte key.
func KeyFromHex(s string) ([32]byte, error) {
	var key [32]byte
	if len(s) != hex.EncodedLen(len(key)) {
		return key, fmt.Errorf("chant: parse key length: %w", ErrBadKeyLength)
	}

	decoded, err := hex.DecodeString(s)
	if err != nil {
		return key, fmt.Errorf("chant: decode key hex: %w", ErrBadKeyHex)
	}
	copy(key[:], decoded)
	return key, nil
}

// GenerateKey returns 32 random bytes from crypto/rand as a hex string.
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("chant: generate key: %w", err)
	}
	return hex.EncodeToString(key), nil
}
