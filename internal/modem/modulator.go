// Package modem implements CHANT's 2-FSK modulation and demodulation.
package modem

import (
	"math"

	"github.com/khelechy/chant/internal/frame"
)

const (
	// DefaultSampleRate is the MVP sample rate.
	DefaultSampleRate = 48000
	// SymbolRate is the MVP symbol rate in baud.
	SymbolRate = 250
	// DefaultSamplesPerSymbol is the number of samples in each symbol window.
	DefaultSamplesPerSymbol = DefaultSampleRate / SymbolRate
	// DefaultF0 is the frequency used for a zero bit.
	DefaultF0 = 1200.0
	// DefaultF1 is the frequency used for a one bit.
	DefaultF1 = 1800.0
	// Amplitude is the modulator output amplitude.
	Amplitude = 0.4
	// EdgeRampFraction applies a small raised-cosine ramp at symbol edges to reduce harsh transitions.
	EdgeRampFraction = 0.1
)

// Modulator emits phase-continuous 2-FSK symbols.
type Modulator struct {
	SampleRate       int
	SamplesPerSymbol int
	F0, F1           float64
	phase            float64
}

// NewModulator constructs a 2-FSK modulator for the provided sample rate.
func NewModulator(sampleRate int) *Modulator {
	if sampleRate <= 0 {
		sampleRate = DefaultSampleRate
	}
	return &Modulator{
		SampleRate:       sampleRate,
		SamplesPerSymbol: sampleRate / SymbolRate,
		F0:               DefaultF0,
		F1:               DefaultF1,
	}
}

// ModulateBits produces phase-continuous 2-FSK samples.
func (m *Modulator) ModulateBits(bits []bool) []float32 {
	if len(bits) == 0 {
		return nil
	}

	samples := make([]float32, 0, len(bits)*m.SamplesPerSymbol)
	rampSamples := int(float64(m.SamplesPerSymbol) * EdgeRampFraction)
	if rampSamples < 1 {
		rampSamples = 1
	}
	for _, bit := range bits {
		frequency := m.F0
		if bit {
			frequency = m.F1
		}
		phaseStep := 2 * math.Pi * frequency / float64(m.SampleRate)
		for i := 0; i < m.SamplesPerSymbol; i++ {
			envelope := edgeEnvelope(i, m.SamplesPerSymbol, rampSamples)
			samples = append(samples, float32(Amplitude*envelope*math.Sin(m.phase)))
			m.phase += phaseStep
			if m.phase >= 2*math.Pi {
				m.phase = math.Mod(m.phase, 2*math.Pi)
			}
		}
	}
	return samples
}

// ModulateBytes unpacks bytes MSB-first then calls ModulateBits.
func (m *Modulator) ModulateBytes(b []byte) []float32 {
	return m.ModulateBits(bytesToBits(b))
}

// ModulatePacket prepends the preamble then modulates framed bytes.
func (m *Modulator) ModulatePacket(framed []byte) []float32 {
	packetBits := append(frame.PreambleBits(), bytesToBits(framed)...)
	return m.ModulateBits(packetBits)
}

func bytesToBits(b []byte) []bool {
	bits := make([]bool, 0, len(b)*8)
	for _, value := range b {
		for shift := 7; shift >= 0; shift-- {
			bits = append(bits, value&(1<<shift) != 0)
		}
	}
	return bits
}

func bitsToBytes(bits []bool) []byte {
	if len(bits) == 0 {
		return nil
	}
	count := len(bits) / 8
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		var value byte
		for bit := 0; bit < 8; bit++ {
			if bits[i*8+bit] {
				value |= 1 << (7 - bit)
			}
		}
		out[i] = value
	}
	return out
}

func edgeEnvelope(index, samplesPerSymbol, rampSamples int) float64 {
	if rampSamples <= 0 || samplesPerSymbol <= 2*rampSamples {
		return 1
	}
	if index < rampSamples {
		position := float64(index+1) / float64(rampSamples+1)
		return 0.5 - 0.5*math.Cos(math.Pi*position)
	}
	if index >= samplesPerSymbol-rampSamples {
		remaining := samplesPerSymbol - index
		position := float64(remaining) / float64(rampSamples+1)
		return 0.5 - 0.5*math.Cos(math.Pi*position)
	}
	return 1
}
