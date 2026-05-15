// Package errs defines shared sentinel errors for the CHANT protocol stack.
package errs

import "errors"

var (
	ErrFECDecode     = errors.New("chant: fec decode failed (too many damaged shards)")
	ErrSyncNotFound  = errors.New("chant: frame sync word not found in samples")
	ErrCRCMismatch   = errors.New("chant: frame crc mismatch")
	ErrInvalidLength = errors.New("chant: frame length field invalid")
	ErrBadKeyLength  = errors.New("chant: key must be 32 bytes (64 hex chars)")
	ErrBadKeyHex     = errors.New("chant: key must be valid hex")
	ErrWAVFormat     = errors.New("chant: wav must be mono 16-bit pcm")
)
