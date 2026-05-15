package chant

import (
	"bytes"
	"crypto/rand"
	"math"
	mathrand "math/rand"
	"path/filepath"
	"testing"
	"testing/quick"

	"github.com/khelechy/chant/internal/wavio"
)

func TestRoundTrip(t *testing.T) {
	key := fixedTestKey()
	cases := []struct {
		name string
		msg  []byte
	}{
		{name: "empty", msg: []byte("")},
		{name: "short", msg: []byte("hello chant")},
		{name: "pangram", msg: []byte("The quick brown fox jumps over the lazy dog")},
		{name: "random", msg: randomBytes(t, 256)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			samples, err := EncodeMessage(key, tc.msg)
			if err != nil {
				t.Fatalf("EncodeMessage() error = %v", err)
			}

			path := filepath.Join(t.TempDir(), tc.name+".wav")
			if err := wavio.WriteWAV(path, samples, 48000); err != nil {
				t.Fatalf("WriteWAV() error = %v", err)
			}

			readSamples, sampleRate, err := wavio.ReadWAV(path)
			if err != nil {
				t.Fatalf("ReadWAV() error = %v", err)
			}
			if sampleRate != 48000 {
				t.Fatalf("sampleRate = %d, want 48000", sampleRate)
			}

			got, err := DecodeMessage(key, readSamples)
			if err != nil {
				t.Fatalf("DecodeMessage() error = %v", err)
			}
			if !bytes.Equal(got, tc.msg) {
				t.Fatalf("DecodeMessage() = %q, want %q", got, tc.msg)
			}
		})
	}
}

func TestRoundTripNoisy(t *testing.T) {
	key := fixedTestKey()
	msg := []byte("CHANT should survive moderate white noise")

	samples, err := EncodeMessage(key, msg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	noisy := addGaussianNoise(samples, 20, 1)
	got, err := DecodeMessage(key, noisy)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("DecodeMessage() = %q, want %q", got, msg)
	}
}

func TestRoundTripWrongKey(t *testing.T) {
	keyA := fixedTestKey()
	keyB := [32]byte{9, 9, 9, 9, 9}

	samples, err := EncodeMessage(keyA, []byte("hello chant"))
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	if _, err := DecodeMessage(keyB, samples); err == nil {
		t.Fatal("DecodeMessage() error = nil, want failure")
	}
}

func TestPropertyRoundTrip(t *testing.T) {
	key := fixedTestKey()
	f := func(b []byte) bool {
		if len(b) == 0 || len(b) > 1024 {
			return true
		}
		samples, err := EncodeMessage(key, b)
		if err != nil {
			return false
		}
		got, err := DecodeMessage(key, samples)
		if err != nil {
			return false
		}
		return bytes.Equal(got, b)
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatal(err)
	}
}

func fixedTestKey() [32]byte {
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}
	return buf
}

func addGaussianNoise(samples []float32, snrDB float64, seed int64) []float32 {
	noisy := make([]float32, len(samples))
	if len(samples) == 0 {
		return noisy
	}

	var power float64
	for _, sample := range samples {
		power += float64(sample * sample)
	}
	power /= float64(len(samples))
	noisePower := power / math.Pow(10, snrDB/10)
	noiseStdDev := math.Sqrt(noisePower)

	rng := mathrand.New(mathrand.NewSource(seed))
	for i, sample := range samples {
		noisy[i] = sample + float32(rng.NormFloat64()*noiseStdDev)
	}
	return noisy
}
