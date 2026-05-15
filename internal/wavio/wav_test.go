package wavio

import (
	"math"
	"path/filepath"
	"testing"
)

func TestWriteReadWAVRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.wav")

	input := make([]float32, 512)
	for i := range input {
		input[i] = float32(0.5 * math.Sin(2*math.Pi*440*float64(i)/48000))
	}

	if err := WriteWAV(path, input, 48000); err != nil {
		t.Fatalf("WriteWAV() error = %v", err)
	}

	got, sampleRate, err := ReadWAV(path)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}
	if sampleRate != 48000 {
		t.Fatalf("sampleRate = %d, want 48000", sampleRate)
	}
	if len(got) != len(input) {
		t.Fatalf("len(samples) = %d, want %d", len(got), len(input))
	}
	for i := range input {
		if math.Abs(float64(got[i]-input[i])) > 2.0/32768.0 {
			t.Fatalf("sample[%d] = %f, want %f within quantization tolerance", i, got[i], input[i])
		}
	}
}
