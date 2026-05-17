// Package modem implements CHANT's 2-FSK modulation and demodulation.
package modem

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/khelechy/chant/internal/errs"
	"github.com/khelechy/chant/internal/frame"
)

const (
	preambleBitsLength   = frame.PreambleSymbols
	syncBitsLength       = 32
	lengthBitsPerField   = 16
	crcBitsLength        = 32
	minimumFrameBitCount = syncBitsLength + (2 * lengthBitsPerField) + crcBitsLength
	maxPreambleBitErrors = 6
	maxSyncBitErrors     = 2
)

// Demodulator recovers framed bytes from 2-FSK samples.
type Demodulator struct {
	SampleRate       int
	SamplesPerSymbol int
	F0, F1           float64
}

// NewDemodulator constructs a 2-FSK demodulator for the provided sample rate.
func NewDemodulator(sampleRate int) *Demodulator {
	if sampleRate <= 0 {
		sampleRate = DefaultSampleRate
	}
	return &Demodulator{
		SampleRate:       sampleRate,
		SamplesPerSymbol: sampleRate / SymbolRate,
		F0:               DefaultF0,
		F1:               DefaultF1,
	}
}

// Demodulate runs Goertzel detection per symbol, finds the sync word, and returns the framed bytes.
func (d *Demodulator) Demodulate(samples []float32) ([]byte, error) {
	for offset := 0; offset < d.SamplesPerSymbol && offset < len(samples); offset++ {
		framed, err := d.demodulateAtOffset(samples, offset)
		if err == nil {
			return framed, nil
		}
	}

	return nil, fmt.Errorf("chant: locate sync word in samples: %w", errs.ErrSyncNotFound)
}

func (d *Demodulator) demodulateAtOffset(samples []float32, offset int) ([]byte, error) {
	if offset < 0 || offset >= len(samples) {
		return nil, fmt.Errorf("chant: invalid symbol offset: %w", errs.ErrInvalidLength)
	}

	bits := d.demodulateBits(samples[offset:])
	preamblePattern := frame.PreambleBits()
	syncPattern := syncWordBits()

	for start := 0; start+preambleBitsLength+minimumFrameBitCount <= len(bits); start++ {
		if !matchesPreambleAndSync(bits, start, preamblePattern, syncPattern) {
			continue
		}

		frameStart := start + preambleBitsLength
		headerBits := bits[frameStart : frameStart+syncBitsLength+(2*lengthBitsPerField)]
		headerBytes := bitsToBytes(headerBits)
		payloadLen := int(binary.BigEndian.Uint16(headerBytes[6:8]))
		totalBits := minimumFrameBitCount + payloadLen*8
		if payloadLen < 0 || frameStart+totalBits > len(bits) {
			continue
		}

		candidate := bitsToBytes(bits[frameStart : frameStart+totalBits])
		if _, _, err := frame.Unframe(candidate); err == nil {
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("chant: locate sync word in samples: %w", errs.ErrSyncNotFound)
}

func (d *Demodulator) demodulateBits(samples []float32) []bool {
	if d.SamplesPerSymbol <= 0 || len(samples) < d.SamplesPerSymbol {
		return nil
	}

	symbolCount := len(samples) / d.SamplesPerSymbol
	bits := make([]bool, 0, symbolCount)
	for symbol := 0; symbol < symbolCount; symbol++ {
		start := symbol * d.SamplesPerSymbol
		window := samples[start : start+d.SamplesPerSymbol]
		g0 := goertzel(window, d.F0, float64(d.SampleRate))
		g1 := goertzel(window, d.F1, float64(d.SampleRate))
		bits = append(bits, g1 > g0)
	}
	return bits
}

func goertzel(samples []float32, targetFreq float64, sampleRate float64) float64 {
	n := float64(len(samples))
	k := 0.5 + n*targetFreq/sampleRate
	omega := 2 * math.Pi * k / n
	cosine := math.Cos(omega)
	coeff := 2 * cosine

	var sPrev float64
	var sPrev2 float64
	for _, sample := range samples {
		s := coeff*sPrev - sPrev2 + float64(sample)
		sPrev2 = sPrev
		sPrev = s
	}

	return sPrev*sPrev + sPrev2*sPrev2 - coeff*sPrev*sPrev2
}

func syncWordBits() []bool {
	var syncBytes [4]byte
	binary.BigEndian.PutUint32(syncBytes[:], frame.SyncWord)
	return bytesToBits(syncBytes[:])
}

func equalBits(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func matchesPreambleAndSync(bits []bool, start int, preamble []bool, sync []bool) bool {
	if start < 0 || start+len(preamble)+len(sync) > len(bits) {
		return false
	}

	preambleErrors := bitErrors(bits[start:start+len(preamble)], preamble)
	if preambleErrors > maxPreambleBitErrors {
		return false
	}

	syncStart := start + len(preamble)
	syncErrors := bitErrors(bits[syncStart:syncStart+len(sync)], sync)
	return syncErrors <= maxSyncBitErrors
}

func bitErrors(a, b []bool) int {
	if len(a) != len(b) {
		return len(a) + len(b)
	}

	errors := 0
	for i := range a {
		if a[i] != b[i] {
			errors++
		}
	}
	return errors
}
