// Package wavio handles WAV file boundaries for CHANT audio samples.
package wavio

import (
	"fmt"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/khelechy/chant/internal/errs"
)

const pcmBitDepth = 16

// WriteWAV writes mono 16-bit PCM at the given sample rate.
// It converts float32 [-1, 1] to int16, clipping at +/-1.
func WriteWAV(path string, samples []float32, sampleRate int) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("chant: create wav file: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("chant: close wav file: %w", closeErr)
		}
	}()

	encoder := wav.NewEncoder(file, sampleRate, pcmBitDepth, 1, 1)
	buffer := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sampleRate,
		},
		SourceBitDepth: pcmBitDepth,
		Data:           make([]int, len(samples)),
	}
	for i, sample := range samples {
		clipped := sample
		if clipped > 1 {
			clipped = 1
		}
		if clipped < -1 {
			clipped = -1
		}
		buffer.Data[i] = int(int16(clipped * 32767))
	}

	if err := encoder.Write(buffer); err != nil {
		return fmt.Errorf("chant: write wav data: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("chant: finalize wav file: %w", err)
	}
	return nil
}

// ReadWAV reads mono 16-bit PCM and returns float32 samples plus sample rate.
func ReadWAV(path string) (samples []float32, sampleRate int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("chant: open wav file: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("chant: close wav file: %w", closeErr)
		}
	}()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("chant: invalid wav file: %w", errs.ErrWAVFormat)
	}
	if decoder.NumChans != 1 || decoder.BitDepth != pcmBitDepth {
		return nil, 0, fmt.Errorf("chant: unsupported wav format: %w", errs.ErrWAVFormat)
	}

	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("chant: read wav pcm data: %w", err)
	}
	if buffer == nil || buffer.Format == nil || buffer.Format.NumChannels != 1 {
		return nil, 0, fmt.Errorf("chant: wav buffer format mismatch: %w", errs.ErrWAVFormat)
	}

	samples = make([]float32, len(buffer.Data))
	for i, value := range buffer.Data {
		samples[i] = float32(value) / 32768
	}
	return samples, buffer.Format.SampleRate, nil
}
