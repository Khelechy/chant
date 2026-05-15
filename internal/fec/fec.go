// Package fec implements shard-based forward error correction for CHANT.
package fec

import (
	"encoding/binary"
	"fmt"

	"github.com/khelechy/chant/internal/errs"
	"github.com/klauspost/reedsolomon"
)

const (
	// DataShards is the number of data shards per FEC block.
	DataShards = 8
	// ParityShards is the number of parity shards per FEC block.
	ParityShards = 4
	// ShardSize is the size of each shard before its CRC-16 trailer.
	ShardSize = 32

	crcSize         = 2
	totalShards     = DataShards + ParityShards
	shardRecordSize = ShardSize + crcSize
	blockDataSize   = DataShards * ShardSize
	blockWireSize   = totalShards * shardRecordSize
	crc16Init       = 0xFFFF
	crc16Poly       = 0x1021
)

// FECEncode pads input to a multiple of DataShards*ShardSize, computes parity
// shards, appends a CRC-16 to every shard, and returns the concatenated shard
// stream along with the original pre-padding length.
//
// The klauspost/reedsolomon package is an erasure-code implementation. For the
// MVP, CHANT uses a CRC-per-shard scheme to turn damaged shards into erasures:
// any shard whose CRC fails at decode time is marked missing and reconstructed.
// This implementation uses CRC-16/CCITT-FALSE for shard checksums. It detects
// and recovers missing-or-damaged shards up to ParityShards per FEC block, but
// it is not a full unknown-position symbol error-correcting code.
func FECEncode(data []byte) (encoded []byte, originalLen int, err error) {
	originalLen = len(data)
	codec, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, 0, fmt.Errorf("chant: create fec encoder: %w", err)
	}

	padded := make([]byte, roundUp(originalLen, blockDataSize))
	copy(padded, data)

	encoded = make([]byte, 0, (len(padded)/blockDataSize)*blockWireSize)
	for offset := 0; offset < len(padded); offset += blockDataSize {
		shards := make([][]byte, totalShards)
		for shardIndex := 0; shardIndex < DataShards; shardIndex++ {
			start := offset + shardIndex*ShardSize
			shard := make([]byte, ShardSize)
			copy(shard, padded[start:start+ShardSize])
			shards[shardIndex] = shard
		}
		for shardIndex := DataShards; shardIndex < totalShards; shardIndex++ {
			shards[shardIndex] = make([]byte, ShardSize)
		}
		if err := codec.Encode(shards); err != nil {
			return nil, 0, fmt.Errorf("chant: encode fec shards: %w", err)
		}

		for _, shard := range shards {
			encoded = append(encoded, shard...)
			var crcBytes [crcSize]byte
			binary.BigEndian.PutUint16(crcBytes[:], crc16(shard))
			encoded = append(encoded, crcBytes[:]...)
		}
	}

	return encoded, originalLen, nil
}

// FECDecode reverses FECEncode, marks CRC-failing shards as erasures, attempts
// reconstruction, and trims any encode-time padding using originalLen.
func FECDecode(encoded []byte, originalLen int) ([]byte, error) {
	if len(encoded)%blockWireSize != 0 {
		return nil, fmt.Errorf("chant: invalid fec block length: %w", errs.ErrInvalidLength)
	}
	if originalLen < 0 {
		return nil, fmt.Errorf("chant: negative original length: %w", errs.ErrInvalidLength)
	}

	codec, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, fmt.Errorf("chant: create fec decoder: %w", err)
	}

	recovered := make([]byte, 0, len(encoded)/blockWireSize*blockDataSize)
	for blockOffset := 0; blockOffset < len(encoded); blockOffset += blockWireSize {
		shards := make([][]byte, totalShards)
		missing := 0

		for shardIndex := 0; shardIndex < totalShards; shardIndex++ {
			recordStart := blockOffset + shardIndex*shardRecordSize
			shard := make([]byte, ShardSize)
			copy(shard, encoded[recordStart:recordStart+ShardSize])
			wantCRC := binary.BigEndian.Uint16(encoded[recordStart+ShardSize : recordStart+shardRecordSize])
			if crc16(shard) != wantCRC {
				missing++
				continue
			}
			shards[shardIndex] = shard
		}

		if missing > 0 {
			if err := codec.Reconstruct(shards); err != nil {
				return nil, fmt.Errorf("chant: reconstruct fec block: %w", errs.ErrFECDecode)
			}
		}

		ok, err := codec.Verify(shards)
		if err != nil {
			return nil, fmt.Errorf("chant: verify fec block: %w", errs.ErrFECDecode)
		}
		if !ok {
			return nil, fmt.Errorf("chant: fec block verification failed: %w", errs.ErrFECDecode)
		}

		for shardIndex := 0; shardIndex < DataShards; shardIndex++ {
			recovered = append(recovered, shards[shardIndex]...)
		}
	}

	if originalLen > len(recovered) {
		return nil, fmt.Errorf("chant: original length exceeds recovered data: %w", errs.ErrInvalidLength)
	}

	return recovered[:originalLen], nil
}

func roundUp(n, multiple int) int {
	if n == 0 {
		return 0
	}
	if n%multiple == 0 {
		return n
	}
	return ((n / multiple) + 1) * multiple
}

func crc16(data []byte) uint16 {
	crc := uint16(crc16Init)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ crc16Poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
