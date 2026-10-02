import 'dart:math';

import 'package:camera/camera.dart';
import 'package:tflite_flutter/tflite_flutter.dart';

/// Thrown when no usable face embedding can be produced.
class FaceEmbeddingException implements Exception {
  final String message;
  const FaceEmbeddingException(this.message);

  @override
  String toString() => 'FaceEmbeddingException: $message';
}

/// One captured frame: its embedding plus the quality score (0..1) assigned
/// at capture time.
class FaceFrame {
  final List<double> embedding;
  final double quality;

  const FaceFrame({required this.embedding, required this.quality});
}

/// Extracts face embeddings on-device.
///
/// The server never sees raw photos — only the vectors produced here are
/// sent to `/face/enroll` and `/face/verify`.
abstract class FaceEmbeddingService {
  /// Returns an L2-normalized embedding for the dominant face in [image].
  Future<List<double>> extractEmbedding(CameraImage image);

  /// Embedding dimensionality (server accepts 128, 192, 512).
  int get dimension;
}

/// Deterministic pseudo-embeddings for development and automated tests.
///
/// Derives a stable unit vector from the frame bytes — it carries NO real
/// biometric signal and must never ship to production.
class StubEmbeddingService implements FaceEmbeddingService {
  @override
  int get dimension => 128;

  @override
  Future<List<double>> extractEmbedding(CameraImage image) async {
    var hash = 0x811c9dc5; // FNV-1a over a sample of the frame bytes
    for (final plane in image.planes) {
      final bytes = plane.bytes;
      final step = (bytes.length ~/ 256).clamp(1, 1 << 30);
      for (var i = 0; i < bytes.length; i += step) {
        hash ^= bytes[i];
        hash = (hash * 0x01000193) & 0xffffffff;
      }
    }
    final rand = Random(hash);
    final vec =
        List<double>.generate(dimension, (_) => rand.nextDouble() * 2 - 1);
    final norm = sqrt(vec.fold<double>(0, (a, b) => a + b * b));
    if (norm == 0) {
      throw const FaceEmbeddingException('Degenerate frame bytes.');
    }
    return vec.map((v) => v / norm).toList(growable: false);
  }
}

/// Real embedding extraction with a MobileFaceNet TFLite model.
///
/// Setup (see mobile/README.md):
/// 1. Place `mobilefacenet.tflite` at `assets/models/mobilefacenet.tflite`.
/// 2. Run with `--dart-define USE_TFLITE_EMBEDDING=true`.
///
/// The model asset is NOT bundled in this repo (license/size) — without it
/// the first call throws [FaceEmbeddingException] with setup instructions.
class TfliteFaceEmbeddingService implements FaceEmbeddingService {
  static const String modelAsset = 'assets/models/mobilefacenet.tflite';
  static const int inputSize = 112;

  Interpreter? _interpreter;

  @override
  int get dimension => 192; // MobileFaceNet output width

  Future<void> _ensureLoaded() async {
    if (_interpreter != null) return;
    try {
      _interpreter = await Interpreter.fromAsset(modelAsset);
    } catch (e) {
      throw FaceEmbeddingException(
        'Model tidak ditemukan di $modelAsset. '
        'Lihat mobile/README.md bagian "Model MobileFaceNet". ($e)',
      );
    }
  }

  @override
  Future<List<double>> extractEmbedding(CameraImage image) async {
    await _ensureLoaded();
    if (image.format.group != ImageFormatGroup.yuv420) {
      throw const FaceEmbeddingException(
          'Hanya format YUV420 (Android) yang didukung implementasi ini.');
    }
    final input = _preprocess(image);
    final output =
        List<double>.filled(dimension, 0.0).reshape([1, dimension]);
    _interpreter!.run(input, output);
    final vec =
        (output[0] as List).map((e) => (e as num).toDouble()).toList();
    final norm = sqrt(vec.fold<double>(0, (a, b) => a + b * b));
    if (norm == 0) {
      throw const FaceEmbeddingException('Model returned a zero vector.');
    }
    return vec.map((v) => v / norm).toList(growable: false);
  }

  /// YUV420 -> RGB -> 112x112 -> float tensor in [-1, 1], batched as
  /// `[1][112][112][3]`.
  List _preprocess(CameraImage image) {
    final rgb = _yuv420ToRgb(image);
    final flat = List<double>.filled(inputSize * inputSize * 3, 0);
    final srcW = image.width;
    final srcH = image.height;
    for (var y = 0; y < inputSize; y++) {
      for (var x = 0; x < inputSize; x++) {
        final sx = (x * srcW / inputSize).floor().clamp(0, srcW - 1);
        final sy = (y * srcH / inputSize).floor().clamp(0, srcH - 1);
        final si = (sy * srcW + sx) * 3;
        final di = (y * inputSize + x) * 3;
        // Nearest-neighbor resize; production should use bilinear + face crop.
        flat[di] = rgb[si] / 127.5 - 1.0;
        flat[di + 1] = rgb[si + 1] / 127.5 - 1.0;
        flat[di + 2] = rgb[si + 2] / 127.5 - 1.0;
      }
    }
    return [
      List.generate(
          inputSize,
          (y) => List.generate(inputSize, (x) {
                final i = (y * inputSize + x) * 3;
                return [flat[i], flat[i + 1], flat[i + 2]];
              })),
    ];
  }

  List<int> _yuv420ToRgb(CameraImage image) {
    final width = image.width;
    final height = image.height;
    final yPlane = image.planes[0];
    final uPlane = image.planes[1];
    final vPlane = image.planes[2];
    final uvRowStride = uPlane.bytesPerRow;
    final uvPixelStride = uPlane.bytesPerPixel ?? 1;
    final rgb = List<int>.filled(width * height * 3, 0);
    for (var y = 0; y < height; y++) {
      for (var x = 0; x < width; x++) {
        final yIndex = y * yPlane.bytesPerRow + x;
        final uvIndex = uvRowStride * (y ~/ 2) + uvPixelStride * (x ~/ 2);
        final yp = yPlane.bytes[yIndex];
        final up = uPlane.bytes[uvIndex];
        final vp = vPlane.bytes[uvIndex];
        final r = (yp + 1.402 * (vp - 128)).round().clamp(0, 255);
        final g =
            (yp - 0.344136 * (up - 128) - 0.714136 * (vp - 128)).round().clamp(0, 255);
        final b = (yp + 1.772 * (up - 128)).round().clamp(0, 255);
        final i = (y * width + x) * 3;
        rgb[i] = r;
        rgb[i + 1] = g;
        rgb[i + 2] = b;
      }
    }
    return rgb;
  }

  void close() {
    _interpreter?.close();
    _interpreter = null;
  }
}

/// Factory wired to `--dart-define USE_TFLITE_EMBEDDING=true`.
FaceEmbeddingService createFaceEmbeddingService({bool useTflite = false}) =>
    useTflite ? TfliteFaceEmbeddingService() : StubEmbeddingService();
