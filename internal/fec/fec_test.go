package fec

import (
	"errors"
	"testing"

	"github.com/khelechy/chant/internal/errs"
)

func TestFECEncodeDecodeRoundTrip(t *testing.T) {
	data := []byte("hello chant")

	encoded, originalLen, err := FECEncode(data)
	if err != nil {
		t.Fatalf("FECEncode() error = %v", err)
	}

	got, err := FECDecode(encoded, originalLen)
	if err != nil {
		t.Fatalf("FECDecode() error = %v", err)
	}

	if string(got) != string(data) {
		t.Fatalf("FECDecode() = %q, want %q", got, data)
	}
}

func TestFECDecodeRecoversSingleShardCorruption(t *testing.T) {
	data := make([]byte, 96)
	for i := range data {
		data[i] = byte(i)
	}

	encoded, originalLen, err := FECEncode(data)
	if err != nil {
		t.Fatalf("FECEncode() error = %v", err)
	}

	encoded[5] ^= 0xFF
	encoded[12] ^= 0x7F

	got, err := FECDecode(encoded, originalLen)
	if err != nil {
		t.Fatalf("FECDecode() error = %v", err)
	}

	for i := range data {
		if got[i] != data[i] {
			t.Fatalf("byte %d = %d, want %d", i, got[i], data[i])
		}
	}
}

func TestFECDecodeFailsBeyondParityThreshold(t *testing.T) {
	data := make([]byte, 128)
	for i := range data {
		data[i] = byte(i)
	}

	encoded, originalLen, err := FECEncode(data)
	if err != nil {
		t.Fatalf("FECEncode() error = %v", err)
	}

	for shardIndex := 0; shardIndex < ParityShards+1; shardIndex++ {
		offset := shardIndex * shardRecordSize
		encoded[offset] ^= 0xFF
	}

	_, err = FECDecode(encoded, originalLen)
	if !errors.Is(err, errs.ErrFECDecode) {
		t.Fatalf("FECDecode() error = %v, want %v", err, errs.ErrFECDecode)
	}
}
