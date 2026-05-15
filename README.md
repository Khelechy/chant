# CHANT

CHANT (Cryptographic Hidden Audio Network Transport) is a Go library and CLI that encodes encrypted text as audio. The output WAV sounds like a two-tone modem chirp or R2-D2-style screech. A receiver reads that audio and decodes it back to the original plaintext.

The MVP is file-based only: it writes and reads 48 kHz mono 16-bit PCM WAV files and does not attempt live microphone capture or speaker playback.

## Install

```bash
go install ./cmd/chant
```

## Project layout

The public API stays at the module root, while implementation details are grouped by concern under `internal/`:

```text
chant/
├── chant.go
├── chant_test.go
├── errors.go
├── cmd/
│   └── chant/
│       └── main.go
└── internal/
    ├── crypto/
    ├── errs/
    ├── fec/
    ├── frame/
    ├── modem/
    └── wavio/
```

## Library usage

```go
package main

import (
	"fmt"

	"github.com/khelechy/chant"
)

func main() {
	keyHex, err := chant.GenerateKey()
	if err != nil {
		panic(err)
	}

	key, err := chant.KeyFromHex(keyHex)
	if err != nil {
		panic(err)
	}

	samples, err := chant.EncodeMessage(key, []byte("hello chant"))
	if err != nil {
		panic(err)
	}

	decoded, err := chant.DecodeMessage(key, samples)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(decoded))
}
```

## CLI usage

Generate a key:

```bash
chant keygen
```

Encode a literal string:

```bash
chant encode --key <HEX> --message "hello chant" --output out.wav
```

Encode from a text file:

```bash
chant encode --key <HEX> --input message.txt --output out.wav
```

Decode to stdout:

```bash
chant decode --key <HEX> --input out.wav
```

Decode to a file:

```bash
chant decode --key <HEX> --input out.wav --output message.txt
```

## Protocol stack

Encoding pipeline:

```text
plaintext
  -> ChaCha20-Poly1305 encrypt
  -> Reed-Solomon FEC with CRC-16 per shard
  -> frame with sync + lengths + CRC32
  -> 2-FSK modulation
  -> mono 16-bit PCM WAV
```

Decoding pipeline:

```text
WAV
  -> 2-FSK demodulation
  -> sync recovery and frame extraction
  -> CRC32 verification
  -> FEC reconstruction
  -> ChaCha20-Poly1305 decrypt
  -> plaintext
```

## Frame structure

Post-FEC, pre-modulation packets are framed as:

```text
[ preamble: 32 alternating symbols 0101... ]
[ sync: 32 bits = 0x1ACFFC1D ]
[ originalLen: 16 bits, big-endian ]
[ payloadLen: 16 bits, big-endian ]
[ payload: FEC-encoded ciphertext bytes ]
[ crc32: 32 bits over originalLen || payloadLen || payload ]
```

On the wire, the payload carried in the frame is:

```text
nonce(12) || ciphertext || tag(16)
```

## Audio parameters

- Sample rate: 48000 Hz
- Channels: mono
- WAV sample format: signed 16-bit PCM
- Symbol rate: 100 baud
- Samples per symbol: 480
- Bit 0 frequency: 1500 Hz
- Bit 1 frequency: 2500 Hz
- Amplitude: 0.6
- Phase continuity: preserved across symbol boundaries

## Limitations

The FEC layer uses `github.com/klauspost/reedsolomon`, which is an erasure code rather than a classical unknown-error Reed-Solomon decoder. For the MVP, CHANT works around that by attaching a CRC-16/CCITT-FALSE checksum to every data and parity shard. At decode time, any shard whose CRC fails is marked missing, and Reed-Solomon reconstruction fills those erasures.

This means the current FEC can recover up to `ParityShards` damaged shards per block when those damaged shards are detectable by CRC, but it is not as robust as a proper channel code designed for noisy symbol errors.

The demodulator also assumes symbol-aligned sample windows for the MVP. It performs sync search in the recovered bitstream but does not yet implement clock recovery for arbitrary live captures.

## Future work

- Live audio I/O
- Chirp modulation
- Better FEC
- Mobile bindings