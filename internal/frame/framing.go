// Package frame implements byte framing for CHANT packets.
package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/khelechy/chant/internal/errs"
)

const (
	// SyncWord is the 32-bit CCSDS sync word used for frame detection.
	SyncWord uint32 = 0x1ACFFC1D
	// PreambleSymbols is the number of alternating preamble symbols.
	PreambleSymbols = 32

	syncSize        = 4
	lengthFieldSize = 2
	crc32Size       = 4
	headerSize      = syncSize + lengthFieldSize + lengthFieldSize
)

// PreambleBits returns the 32-bit alternating preamble pattern.
func PreambleBits() []bool {
	bits := make([]bool, PreambleSymbols)
	for i := range bits {
		bits[i] = i%2 == 1
	}
	return bits
}

// Frame wraps the FEC-encoded payload with sync, lengths, and CRC32.
func Frame(payload []byte, originalLen uint16) []byte {
	framed := make([]byte, 0, headerSize+len(payload)+crc32Size)
	var syncBytes [syncSize]byte
	binary.BigEndian.PutUint32(syncBytes[:], SyncWord)
	framed = append(framed, syncBytes[:]...)

	var lenBytes [lengthFieldSize]byte
	binary.BigEndian.PutUint16(lenBytes[:], originalLen)
	framed = append(framed, lenBytes[:]...)
	binary.BigEndian.PutUint16(lenBytes[:], uint16(len(payload)))
	framed = append(framed, lenBytes[:]...)
	framed = append(framed, payload...)

	crc := crc32.ChecksumIEEE(framed[syncSize:])
	var crcBytes [crc32Size]byte
	binary.BigEndian.PutUint32(crcBytes[:], crc)
	framed = append(framed, crcBytes[:]...)
	return framed
}

// Unframe verifies sync, CRC, and lengths, returning the payload and original length.
func Unframe(framed []byte) (payload []byte, originalLen uint16, err error) {
	var syncBytes [syncSize]byte
	binary.BigEndian.PutUint32(syncBytes[:], SyncWord)

	syncOffset := bytes.Index(framed, syncBytes[:])
	if syncOffset < 0 {
		return nil, 0, fmt.Errorf("chant: locate sync word: %w", errs.ErrSyncNotFound)
	}
	framed = framed[syncOffset:]

	if len(framed) < headerSize+crc32Size {
		return nil, 0, fmt.Errorf("chant: frame too short: %w", errs.ErrInvalidLength)
	}

	originalLen = binary.BigEndian.Uint16(framed[syncSize : syncSize+lengthFieldSize])
	payloadLen := binary.BigEndian.Uint16(framed[syncSize+lengthFieldSize : headerSize])
	expectedLen := headerSize + int(payloadLen) + crc32Size
	if expectedLen > len(framed) {
		return nil, 0, fmt.Errorf("chant: truncated frame payload: %w", errs.ErrInvalidLength)
	}

	frameBody := framed[syncSize : headerSize+int(payloadLen)]
	wantCRC := binary.BigEndian.Uint32(framed[headerSize+int(payloadLen) : expectedLen])
	if crc32.ChecksumIEEE(frameBody) != wantCRC {
		return nil, 0, fmt.Errorf("chant: verify frame crc: %w", errs.ErrCRCMismatch)
	}

	payload = make([]byte, payloadLen)
	copy(payload, framed[headerSize:headerSize+int(payloadLen)])
	return payload, originalLen, nil
}
