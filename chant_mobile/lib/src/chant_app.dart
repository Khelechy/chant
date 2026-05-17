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
    const panel = Color(0xFF141A12);
    const surface = Color(0xFF1E281A);
    const chrome = Color(0xFF32402A);
    const accent = Color(0xFF9FB36B);
    const warning = Color(0xFFE1B866);
    const text = Color(0xFFD6DEC8);

    return MaterialApp(
      title: 'chant_mobile',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: const ColorScheme.dark(
          primary: accent,
          secondary: warning,
          surface: surface,
          onPrimary: panel,
          onSecondary: panel,
          onSurface: text,
          error: Color(0xFFE27D60),
        ),
        scaffoldBackgroundColor: panel,
        dividerColor: chrome,
        useMaterial3: true,
        inputDecorationTheme: InputDecorationTheme(
          filled: true,
          fillColor: const Color(0xFF11160F),
          labelStyle: const TextStyle(color: text),
          hintStyle: TextStyle(color: text.withValues(alpha: 0.55)),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: chrome),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: chrome),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: accent, width: 1.4),
          ),
        ),
        filledButtonTheme: FilledButtonThemeData(
          style: FilledButton.styleFrom(
            backgroundColor: accent,
            foregroundColor: panel,
            disabledBackgroundColor: chrome,
            disabledForegroundColor: text.withValues(alpha: 0.5),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            textStyle: const TextStyle(fontWeight: FontWeight.w700, letterSpacing: 0.8),
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 18),
          ),
        ),
        outlinedButtonTheme: OutlinedButtonThemeData(
          style: OutlinedButton.styleFrom(
            foregroundColor: warning,
            side: const BorderSide(color: chrome),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            textStyle: const TextStyle(fontWeight: FontWeight.w700, letterSpacing: 0.8),
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 18),
          ),
        ),
        textTheme: const TextTheme(
          headlineMedium: TextStyle(
            fontSize: 30,
            fontWeight: FontWeight.w800,
            letterSpacing: 1.2,
            color: text,
          ),
          titleMedium: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.3,
            color: warning,
          ),
          bodyMedium: TextStyle(
            fontSize: 14,
            height: 1.45,
            color: text,
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
    final fileLabel = _wavPath == null ? 'No WAV file selected yet.' : _basename(_wavPath!);
    final isWide = MediaQuery.of(context).size.width >= 920;

    return Scaffold(
      body: SafeArea(
        child: Container(
          width: double.infinity,
          height: double.infinity,
          decoration: const BoxDecoration(
            color: Color(0xFF141A12),
          ),
          child: Stack(
            children: [
              const Positioned.fill(child: _TacticalBackdrop()),
              SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(18, 18, 18, 28),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 640),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _Header(status: _status, isBusy: _isBusy, isRecording: _isRecording),
                      const SizedBox(height: 18),
                      if (isWide)
                        Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(
                              flex: 11,
                              child: _ControlPanel(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    const _SectionTitle(title: 'Connection uplink'),
                                    TextField(
                                      controller: _serverController,
                                      decoration: const InputDecoration(
                                        labelText: 'Server URL',
                                        hintText: 'http://10.0.2.2:8080/v1/decode',
                                      ),
                                    ),
                                    const SizedBox(height: 12),
                                    TextField(
                                      controller: _keyController,
                                      decoration: const InputDecoration(
                                        labelText: 'CHANT key (hex)',
                                        hintText: '64-character hex key',
                                      ),
                                      autocorrect: false,
                                      enableSuggestions: false,
                                    ),
                                    const SizedBox(height: 18),
                                    const _SectionTitle(title: 'Acquisition'),
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
                                    _DataStrip(label: 'Source file', value: fileLabel),
                                    const SizedBox(height: 18),
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
                                        label: Text(_isBusy ? 'Processing link...' : 'Send to server and decrypt'),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                            const SizedBox(width: 18),
                            Expanded(
                              flex: 9,
                              child: _ResultsPanel(
                                serverResponse: _serverResponse,
                                plainText: _plainText,
                              ),
                            ),
                          ],
                        )
                      else ...[
                        _ControlPanel(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const _SectionTitle(title: 'Connection uplink'),
                              TextField(
                                controller: _serverController,
                                decoration: const InputDecoration(
                                  labelText: 'Server URL',
                                  hintText: 'http://10.0.2.2:8080/v1/decode',
                                ),
                              ),
                              const SizedBox(height: 12),
                              TextField(
                                controller: _keyController,
                                decoration: const InputDecoration(
                                  labelText: 'CHANT key (hex)',
                                  hintText: '64-character hex key',
                                ),
                                autocorrect: false,
                                enableSuggestions: false,
                              ),
                              const SizedBox(height: 18),
                              const _SectionTitle(title: 'Acquisition'),
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
                              _DataStrip(label: 'Source file', value: fileLabel),
                              const SizedBox(height: 18),
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
                                  label: Text(_isBusy ? 'Processing link...' : 'Send to server and decrypt'),
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 18),
                        _ResultsPanel(
                          serverResponse: _serverResponse,
                          plainText: _plainText,
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ],
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
      child: Row(
        children: [
          Container(
            width: 10,
            height: 10,
            decoration: const BoxDecoration(
              color: Color(0xFFE1B866),
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 10),
          Text(title.toUpperCase(), style: Theme.of(context).textTheme.titleMedium),
        ],
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.status, required this.isBusy, required this.isRecording});

  final String? status;
  final bool isBusy;
  final bool isRecording;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final indicatorColor = isRecording
        ? const Color(0xFFE27D60)
        : isBusy
            ? const Color(0xFFE1B866)
            : const Color(0xFF9FB36B);
    final indicatorLabel = isRecording
        ? 'Recording'
        : isBusy
            ? 'Transmitting'
            : 'Ready';

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: const Color(0xFF1A2117).withValues(alpha: 0.92),
        border: Border.all(color: const Color(0xFF3D4B30)),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Text('CHANT // FIELD RECEIVER', style: theme.textTheme.headlineMedium),
              ),
              const SizedBox(width: 12),
              _StatusPill(label: indicatorLabel, color: indicatorColor),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            'Server-backed CHANT listener',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: const Color(0xFFE1B866),
              fontWeight: FontWeight.w700,
              letterSpacing: 0.9,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Capture or import a WAV signal, push it to the decode endpoint, and recover plaintext locally using the device-held key.',
            style: theme.textTheme.bodyMedium,
          ),
          if (status != null) ...[
            const SizedBox(height: 14),
            _DataStrip(label: 'Status', value: status!),
          ],
        ],
      ),
    );
  }
}

class _ControlPanel extends StatelessWidget {
  const _ControlPanel({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: const Color(0xFF1A2117).withValues(alpha: 0.9),
        border: Border.all(color: const Color(0xFF3D4B30)),
        borderRadius: BorderRadius.circular(10),
      ),
      child: child,
    );
  }
}

class _ResultsPanel extends StatelessWidget {
  const _ResultsPanel({required this.serverResponse, required this.plainText});

  final _ServerDecodeResponse? serverResponse;
  final String? plainText;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return _ControlPanel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _SectionTitle(title: 'Telemetry'),
          if (serverResponse == null)
            Text(
              'Awaiting decode response. Telemetry will populate after a successful upload.',
              style: theme.textTheme.bodyMedium,
            )
          else ...[
            _DataStrip(label: 'Sample rate', value: '${serverResponse!.sampleRate} Hz'),
            const SizedBox(height: 10),
            _DataStrip(label: 'Sample count', value: '${serverResponse!.sampleCount}'),
            const SizedBox(height: 10),
            _DataStrip(label: 'Encrypted bytes', value: '${serverResponse!.encryptedBytes}'),
            const SizedBox(height: 10),
            _DataStrip(label: 'File', value: serverResponse!.filename ?? 'uploaded.wav'),
          ],
          const SizedBox(height: 20),
          const _SectionTitle(title: 'Recovered plaintext'),
          Container(
            width: double.infinity,
            constraints: const BoxConstraints(minHeight: 220),
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: const Color(0xFF10150E),
              border: Border.all(color: const Color(0xFF3D4B30)),
              borderRadius: BorderRadius.circular(8),
            ),
            child: plainText == null
                ? Text(
                    'No plaintext recovered yet.',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: const Color(0xFFD6DEC8).withValues(alpha: 0.68),
                    ),
                  )
                : SelectableText(
                    plainText!,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontFamily: 'monospace',
                      fontSize: 15,
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

class _DataStrip extends StatelessWidget {
  const _DataStrip({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: const Color(0xFF10150E),
        border: Border.all(color: const Color(0xFF3D4B30)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: RichText(
        text: TextSpan(
          style: theme.textTheme.bodyMedium,
          children: [
            TextSpan(
              text: '${label.toUpperCase()}  ',
              style: const TextStyle(
                color: Color(0xFFE1B866),
                fontWeight: FontWeight.w700,
                letterSpacing: 1,
              ),
            ),
            TextSpan(text: value),
          ],
        ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.16),
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 8),
          Text(
            label.toUpperCase(),
            style: TextStyle(
              color: color,
              fontWeight: FontWeight.w800,
              letterSpacing: 1,
            ),
          ),
        ],
      ),
    );
  }
}

class _TacticalBackdrop extends StatelessWidget {
  const _TacticalBackdrop();

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      painter: _BackdropPainter(),
    );
  }
}

class _BackdropPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final basePaint = Paint()..color = const Color(0xFF141A12);
    canvas.drawRect(Offset.zero & size, basePaint);

    final gridPaint = Paint()
      ..color = const Color(0xFF2B3425).withValues(alpha: 0.32)
      ..strokeWidth = 1;
    const spacing = 34.0;
    for (double x = 0; x <= size.width; x += spacing) {
      canvas.drawLine(Offset(x, 0), Offset(x, size.height), gridPaint);
    }
    for (double y = 0; y <= size.height; y += spacing) {
      canvas.drawLine(Offset(0, y), Offset(size.width, y), gridPaint);
    }

    final bandPaint = Paint()..color = const Color(0xFF89A55A).withValues(alpha: 0.08);
    canvas.drawRect(Rect.fromLTWH(0, size.height * 0.12, size.width, size.height * 0.1), bandPaint);
    canvas.drawRect(Rect.fromLTWH(0, size.height * 0.76, size.width, size.height * 0.08), bandPaint);

    final ringPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.4
      ..color = const Color(0xFFE1B866).withValues(alpha: 0.12);
    canvas.drawCircle(Offset(size.width * 0.86, size.height * 0.18), 120, ringPaint);
    canvas.drawCircle(Offset(size.width * 0.14, size.height * 0.82), 90, ringPaint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
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