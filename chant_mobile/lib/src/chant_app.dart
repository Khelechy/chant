import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';

class ChantMobileApp extends StatelessWidget {
  const ChantMobileApp({super.key});

  @override
  Widget build(BuildContext context) {
    const base = Color(0xFF102A43);
    const accent = Color(0xFFF0B429);

    return MaterialApp(
      title: 'chant_mobile',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: accent, brightness: Brightness.light),
        scaffoldBackgroundColor: const Color(0xFFF3EFE4),
        useMaterial3: true,
        textTheme: const TextTheme(
          headlineMedium: TextStyle(
            fontSize: 28,
            fontWeight: FontWeight.w800,
            color: base,
          ),
          titleMedium: TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.w700,
            color: base,
          ),
          bodyMedium: TextStyle(
            fontSize: 14,
            height: 1.45,
            color: Color(0xFF243B53),
          ),
        ),
      ),
      home: const ChantListenerPage(),
    );
  }
}

class ChantListenerPage extends StatefulWidget {
  const ChantListenerPage({super.key});

  @override
  State<ChantListenerPage> createState() => _ChantListenerPageState();
}

class _ChantListenerPageState extends State<ChantListenerPage> {
  final AudioRecorder _recorder = AudioRecorder();
  final TextEditingController _serverController = TextEditingController(
    text: 'https://9rf91q83-8080.uks1.devtunnels.ms/v1/decode',
  );
  final TextEditingController _keyController = TextEditingController();

  bool _isBusy = false;
  bool _isRecording = false;
  String? _wavPath;
  String? _status;
  String? _plainText;
  _ServerDecodeResponse? _serverResponse;

  @override
  void dispose() {
    _serverController.dispose();
    _keyController.dispose();
    _recorder.dispose();
    super.dispose();
  }

  Future<void> _toggleRecording() async {
    if (_isBusy) {
      return;
    }

    try {
      if (_isRecording) {
        final savedPath = await _recorder.stop();
        if (!mounted) {
          return;
        }
        setState(() {
          _isRecording = false;
          _wavPath = savedPath;
          _status = savedPath == null ? 'Recording stopped without a file.' : 'Recorded WAV ready for upload.';
        });
        return;
      }

      final hasPermission = await _recorder.hasPermission();
      if (!hasPermission) {
        throw const FormatException('Microphone permission was denied.');
      }

      final wavSupported = await _recorder.isEncoderSupported(AudioEncoder.wav);
      if (!wavSupported) {
        throw const FormatException('This device does not support direct WAV recording.');
      }

      final directory = await getTemporaryDirectory();
      final path = '${directory.path}/chant_${DateTime.now().millisecondsSinceEpoch}.wav';
      await _recorder.start(
        const RecordConfig(
          encoder: AudioEncoder.wav,
          sampleRate: 48000,
          numChannels: 1,
        ),
        path: path,
      );

      if (!mounted) {
        return;
      }
      setState(() {
        _isRecording = true;
        _wavPath = path;
        _status = 'Recording to WAV...';
        _plainText = null;
        _serverResponse = null;
      });
    } on Exception catch (error) {
      _setError(error.toString());
    }
  }

  Future<void> _pickWavFile() async {
    if (_isBusy || _isRecording) {
      return;
    }

    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: const ['wav'],
    );
    if (!mounted || result == null || result.files.isEmpty) {
      return;
    }

    setState(() {
      _wavPath = result.files.single.path;
      _status = _wavPath == null ? 'No WAV selected.' : 'WAV selected and ready for upload.';
      _plainText = null;
      _serverResponse = null;
    });
  }

  Future<void> _sendToServer() async {
    if (_isBusy) {
      return;
    }

    final wavPath = _wavPath;
    if (wavPath == null || wavPath.isEmpty) {
      _setError('Select or record a WAV file first.');
      return;
    }

    final keyHex = _keyController.text.trim();
    final serverUrl = _serverController.text.trim();
    if (serverUrl.isEmpty) {
      _setError('Enter the CHANT server URL first.');
      return;
    }

    setState(() {
      _isBusy = true;
      _status = 'Uploading WAV to the CHANT server...';
      _plainText = null;
      _serverResponse = null;
    });

    try {
      final response = await _uploadWav(Uri.parse(serverUrl), wavPath);
      final plaintext = await _decryptBlobLocally(
        keyHex: keyHex,
        encryptedMessageBase64: response.encryptedMessageBase64,
      );

      if (!mounted) {
        return;
      }
      setState(() {
        _isBusy = false;
        _serverResponse = response;
        _plainText = plaintext;
        _status = 'Upload complete. The server demodulated the WAV and the phone decrypted the blob locally.';
      });
    } on Exception catch (error) {
      _setError(error.toString());
    }
  }

  Future<_ServerDecodeResponse> _uploadWav(Uri endpoint, String wavPath) async {
    final request = http.MultipartRequest('POST', _normalizeEndpoint(endpoint));
    request.files.add(await http.MultipartFile.fromPath('audio', wavPath));

    final streamed = await request.send();
    final response = await http.Response.fromStream(streamed);
    final body = jsonDecode(response.body) as Map<String, dynamic>;

    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message = body['error'] as String? ?? 'Server request failed with ${response.statusCode}.';
      throw FormatException(message);
    }

    return _ServerDecodeResponse.fromJson(body);
  }

  Future<String> _decryptBlobLocally({
    required String keyHex,
    required String encryptedMessageBase64,
  }) async {
    final keyBytes = _parseHexKey(keyHex);
    final blob = base64Decode(encryptedMessageBase64);
    if (blob.length < 28) {
      throw const FormatException('Encrypted message is too short to decrypt.');
    }

    final nonce = blob.sublist(0, 12);
    final cipherText = blob.sublist(12, blob.length - 16);
    final tag = blob.sublist(blob.length - 16);
    final algorithm = Chacha20.poly1305Aead();
    final secretBox = SecretBox(cipherText, nonce: nonce, mac: Mac(tag));
    final plaintextBytes = await algorithm.decrypt(
      secretBox,
      secretKey: SecretKey(keyBytes),
    );
    return utf8.decode(plaintextBytes);
  }

  Uint8List _parseHexKey(String value) {
    final trimmed = value.trim();
    final hexPattern = RegExp(r'^[0-9a-fA-F]{64}$');
    if (!hexPattern.hasMatch(trimmed)) {
      throw const FormatException('Key must be a 64-character hex string.');
    }

    final bytes = Uint8List(32);
    for (var index = 0; index < 32; index++) {
      bytes[index] = int.parse(trimmed.substring(index * 2, index * 2 + 2), radix: 16);
    }
    return bytes;
  }

  Uri _normalizeEndpoint(Uri input) {
    if (input.path.isEmpty || input.path == '/') {
      return input.replace(path: '/v1/decode');
    }
    return input;
  }

  void _setError(String message) {
    if (!mounted) {
      return;
    }
    setState(() {
      _isBusy = false;
      _isRecording = false;
      _status = message;
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fileLabel = _wavPath == null ? 'No WAV file selected yet.' : _basename(_wavPath!);

    return Scaffold(
      body: DecoratedBox(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            colors: [Color(0xFF102A43), Color(0xFF243B53), Color(0xFFF0B429)],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        child: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(20),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 720),
                child: Card(
                  elevation: 12,
                  color: const Color(0xFFF7F3E9),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(28)),
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Server-backed CHANT listener', style: theme.textTheme.headlineMedium),
                        const SizedBox(height: 8),
                        Text(
                          'Record or select a WAV, send it to the CHANT server for demodulation, and decrypt the returned encrypted blob locally on the phone.',
                          style: theme.textTheme.bodyMedium,
                        ),
                        const SizedBox(height: 24),
                        const _SectionTitle(title: 'Connection'),
                        TextField(
                          controller: _serverController,
                          decoration: const InputDecoration(
                            labelText: 'Server URL',
                            hintText: 'http://10.0.2.2:8080/v1/decode',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: _keyController,
                          decoration: const InputDecoration(
                            labelText: 'CHANT key (hex)',
                            hintText: '64-character hex key',
                            border: OutlineInputBorder(),
                          ),
                          autocorrect: false,
                          enableSuggestions: false,
                        ),
                        const SizedBox(height: 20),
                        const _SectionTitle(title: 'WAV source'),
                        Row(
                          children: [
                            Expanded(
                              child: FilledButton.icon(
                                onPressed: _isBusy ? null : _toggleRecording,
                                icon: Icon(_isRecording ? Icons.stop_circle_outlined : Icons.mic_none_outlined),
                                label: Text(_isRecording ? 'Stop recording' : 'Record WAV'),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: OutlinedButton.icon(
                                onPressed: _isBusy || _isRecording ? null : _pickWavFile,
                                icon: const Icon(Icons.audio_file_outlined),
                                label: const Text('Pick WAV'),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Text(fileLabel, style: theme.textTheme.titleMedium),
                        const SizedBox(height: 20),
                        SizedBox(
                          width: double.infinity,
                          child: FilledButton.icon(
                            onPressed: _isBusy || _isRecording ? null : _sendToServer,
                            icon: _isBusy
                                ? const SizedBox(
                                    width: 16,
                                    height: 16,
                                    child: CircularProgressIndicator(strokeWidth: 2),
                                  )
                                : const Icon(Icons.cloud_upload_outlined),
                            label: Text(_isBusy ? 'Working...' : 'Send to server and decrypt'),
                          ),
                        ),
                        if (_status != null) ...[
                          const SizedBox(height: 16),
                          Text(_status!, style: theme.textTheme.bodyMedium),
                        ],
                        if (_serverResponse != null) ...[
                          const SizedBox(height: 20),
                          const _SectionTitle(title: 'Server response'),
                          Text(
                            'Sample rate: ${_serverResponse!.sampleRate} Hz\n'
                            'Samples: ${_serverResponse!.sampleCount}\n'
                            'Encrypted bytes: ${_serverResponse!.encryptedBytes}\n'
                            'File: ${_serverResponse!.filename ?? 'uploaded.wav'}',
                            style: theme.textTheme.bodyMedium,
                          ),
                        ],
                        if (_plainText != null) ...[
                          const SizedBox(height: 20),
                          const _SectionTitle(title: 'Decrypted text'),
                          DecoratedBox(
                            decoration: BoxDecoration(
                              color: const Color(0xCCFFFFFF),
                              borderRadius: BorderRadius.circular(18),
                            ),
                            child: Padding(
                              padding: const EdgeInsets.all(16),
                              child: SelectableText(
                                _plainText!,
                                style: theme.textTheme.bodyMedium?.copyWith(
                                  fontFamily: 'monospace',
                                  fontSize: 15,
                                ),
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  String _basename(String path) {
    final normalized = path.replaceAll('\\', '/');
    final parts = normalized.split('/').where((part) => part.isNotEmpty).toList();
    if (parts.isEmpty) {
      return path;
    }
    return parts.last;
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Text(title, style: Theme.of(context).textTheme.titleMedium),
    );
  }
}

class _ServerDecodeResponse {
  _ServerDecodeResponse({
    required this.encryptedMessageBase64,
    required this.sampleRate,
    required this.sampleCount,
    required this.encryptedBytes,
    this.filename,
  });

  factory _ServerDecodeResponse.fromJson(Map<String, dynamic> json) {
    return _ServerDecodeResponse(
      encryptedMessageBase64: json['encryptedMessageBase64'] as String,
      sampleRate: json['sampleRate'] as int,
      sampleCount: json['sampleCount'] as int,
      encryptedBytes: json['encryptedBytes'] as int,
      filename: json['filename'] as String?,
    );
  }

  final String encryptedMessageBase64;
  final int sampleRate;
  final int sampleCount;
  final int encryptedBytes;
  final String? filename;
}