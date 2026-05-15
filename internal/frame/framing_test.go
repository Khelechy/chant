package frame

import (
	"errors"
	"testing"

	"github.com/khelechy/chant/internal/errs"
)

func TestFrameUnframeRoundTrip(t *testing.T) {
	payload := []byte("hello chant")
	framed := Frame(payload, 7)

	gotPayload, gotOriginalLen, err := Unframe(framed)
	if err != nil {
		t.Fatalf("Unframe() error = %v", err)
	}

	if string(gotPayload) != string(payload) {
		t.Fatalf("payload = %q, want %q", gotPayload, payload)
	}
	if gotOriginalLen != 7 {
		t.Fatalf("originalLen = %d, want 7", gotOriginalLen)
	}
}

func TestUnframeCRCMismatch(t *testing.T) {
	framed := Frame([]byte("hello chant"), 11)
	framed[len(framed)-1] ^= 0xFF

	_, _, err := Unframe(framed)
	if !errors.Is(err, errs.ErrCRCMismatch) {
		t.Fatalf("Unframe() error = %v, want %v", err, errs.ErrCRCMismatch)
	}
}

func TestUnframeTruncated(t *testing.T) {
	framed := Frame([]byte("hello chant"), 11)
	framed = framed[:len(framed)-2]

	_, _, err := Unframe(framed)
	if !errors.Is(err, errs.ErrInvalidLength) {
		t.Fatalf("Unframe() error = %v, want %v", err, errs.ErrInvalidLength)
	}
}

func TestUnframeSyncAtOffset(t *testing.T) {
	framed := append([]byte{0x00, 0x01, 0x02}, Frame([]byte("hello chant"), 11)...)

	gotPayload, gotOriginalLen, err := Unframe(framed)
	if err != nil {
		t.Fatalf("Unframe() error = %v", err)
	}

	if string(gotPayload) != "hello chant" {
		t.Fatalf("payload = %q, want %q", gotPayload, "hello chant")
	}
	if gotOriginalLen != 11 {
		t.Fatalf("originalLen = %d, want 11", gotOriginalLen)
	}
}

func TestUnframeMissingSync(t *testing.T) {
	_, _, err := Unframe([]byte{0x00, 0x01, 0x02})
	if !errors.Is(err, errs.ErrSyncNotFound) {
		t.Fatalf("Unframe() error = %v, want %v", err, errs.ErrSyncNotFound)
	}
}
