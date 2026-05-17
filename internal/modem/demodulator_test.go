package modem

import (
	"bytes"
	"testing"

	"github.com/khelechy/chant/internal/frame"
)

func TestDemodulateBitsRecoversOriginalBits(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	demod := NewDemodulator(DefaultSampleRate)
	input := []byte{0xA5, 0x5A, 0xFF, 0x00}

	wantBits := bytesToBits(input)
	samples := mod.ModulateBits(wantBits)
	gotBits := demod.demodulateBits(samples)

	if len(gotBits) != len(wantBits) {
		t.Fatalf("len(bits) = %d, want %d", len(gotBits), len(wantBits))
	}
	for i := range wantBits {
		if gotBits[i] != wantBits[i] {
			t.Fatalf("bit[%d] = %v, want %v", i, gotBits[i], wantBits[i])
		}
	}
}

func TestDemodulatePacketExtractsFrame(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	demod := NewDemodulator(DefaultSampleRate)
	framed := frame.Frame([]byte("hello chant"), 11)

	samples := mod.ModulatePacket(framed)
	got, err := demod.Demodulate(samples)
	if err != nil {
		t.Fatalf("Demodulate() error = %v", err)
	}

	if !bytes.Equal(got, framed) {
		t.Fatalf("Demodulate() = %x, want %x", got, framed)
	}
}

func TestDemodulatePacketWithLeadingSampleOffset(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	demod := NewDemodulator(DefaultSampleRate)
	framed := frame.Frame([]byte("hello chant"), 11)

	samples := mod.ModulatePacket(framed)
	offsetSamples := append(make([]float32, 73), samples...)

	got, err := demod.Demodulate(offsetSamples)
	if err != nil {
		t.Fatalf("Demodulate() error = %v", err)
	}

	if !bytes.Equal(got, framed) {
		t.Fatalf("Demodulate() = %x, want %x", got, framed)
	}
}

func TestDemodulatePacketSkipsFalseSyncWithBadCRC(t *testing.T) {
	mod := NewModulator(DefaultSampleRate)
	demod := NewDemodulator(DefaultSampleRate)
	want := frame.Frame([]byte("hello chant"), 11)

	corrupted := append([]byte(nil), want...)
	corrupted[len(corrupted)-1] ^= 0x01

	bits := append(frame.PreambleBits(), bytesToBits(corrupted)...)
	bits = append(bits, frame.PreambleBits()...)
	bits = append(bits, bytesToBits(want)...)
	samples := mod.ModulateBits(bits)

	got, err := demod.Demodulate(samples)
	if err != nil {
		t.Fatalf("Demodulate() error = %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("Demodulate() = %x, want %x", got, want)
	}
}
