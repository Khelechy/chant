// Package chant exposes the public CHANT encoding and decoding API.
package chant

import "github.com/khelechy/chant/internal/errs"

var (
	// ErrFECDecode reports that too many FEC shards were damaged to reconstruct.
	ErrFECDecode = errs.ErrFECDecode
	// ErrSyncNotFound reports that the demodulator could not locate a frame sync word.
	ErrSyncNotFound = errs.ErrSyncNotFound
	// ErrCRCMismatch reports that a framed packet failed its CRC validation.
	ErrCRCMismatch = errs.ErrCRCMismatch
	// ErrInvalidLength reports invalid frame or FEC lengths.
	ErrInvalidLength = errs.ErrInvalidLength
	// ErrBadKeyLength reports that a key hex string is not 64 characters long.
	ErrBadKeyLength = errs.ErrBadKeyLength
	// ErrBadKeyHex reports that a key hex string is not valid hexadecimal.
	ErrBadKeyHex = errs.ErrBadKeyHex
	// ErrWAVFormat reports that a WAV file is not mono 16-bit PCM.
	ErrWAVFormat = errs.ErrWAVFormat
)
