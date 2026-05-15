package modem

import (
	"math"
	"testing"
)

func TestModulateBitsOutputLength(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	bits := []bool{false, true, false, true, true}

	samples := mod.ModulateBits(bits)
	if len(samples) != len(bits)*mod.SamplesPerSymbol {
		t.Fatalf("len(samples) = %d, want %d", len(samples), len(bits)*mod.SamplesPerSymbol)
	}
}

func TestModulateBitsAmplitudeBounded(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	samples := mod.ModulateBits([]bool{false, true, false, true, true})

	for i, sample := range samples {
		if math.Abs(float64(sample)) > Amplitude+1e-6 {
			t.Fatalf("sample[%d] = %f, want |sample| <= %f", i, sample, Amplitude)
		}
	}
}

func TestModulateBitsPhaseContinuous(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	samples := mod.ModulateBits([]bool{false, false, true, true, false, true})

	for i := 1; i < len(samples); i++ {
		delta := math.Abs(float64(samples[i] - samples[i-1]))
		if delta > 0.25 {
			t.Fatalf("adjacent delta[%d] = %f, want <= 0.25", i, delta)
		}
	}
}
