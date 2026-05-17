# chant_mobile

Flutter client for CHANT's server-backed listener flow.

The app does three things:

- records mono WAV audio on-device or selects an existing WAV file
- uploads that WAV to a CHANT server REST endpoint for demodulation and FEC recovery
- decrypts the returned encrypted CHANT blob locally with the user's key

The app never sends the CHANT key to the server.

## Server API

Start the Go endpoint from the repository root:

```bash
go run ./cmd/chant_server --addr :8080
```

The mobile app sends a multipart `POST` request to `/v1/decode` with a single file field named `audio`.

You can verify the server before opening the app:

```bash
curl http://127.0.0.1:8080/healthz
```

Successful responses look like this:

```json
{
	"encryptedMessageBase64": "...",
	"filename": "capture.wav",
	"sampleRate": 48000,
	"sampleCount": 92160,
	"encryptedBytes": 140
}
```

The server never receives the CHANT key. It only returns the encrypted CHANT blob (`nonce || ciphertext || tag`) after demodulation, framing, and FEC recovery.

## Mobile usage

1. Start the CHANT server.
2. Open the app and set the server URL.
3. Paste the 64-character CHANT key.
4. Record a WAV or pick one from storage.
5. Tap `Send to server and decrypt`.

Recommended server URLs:

- Android emulator: `http://10.0.2.2:8080/v1/decode`
- iOS simulator: `http://127.0.0.1:8080/v1/decode`
- Physical phone on the same network: `http://<your-computer-lan-ip>:8080/v1/decode`
- Remote/public server or tunnel: `https://<your-host>/v1/decode`

If you enter only a base host such as `http://10.0.2.2:8080`, the app automatically expands it to `/v1/decode`.

## Android

From the `chant_mobile/` directory:

```bash
flutter pub get
flutter run
```

If Flutter does not detect your device or emulator, check:

```bash
flutter devices
```

On Android, the app requests microphone access for WAV recording. File picking also works if you already have a WAV to test with.

## Flutter

```bash
flutter pub get
flutter run
```
