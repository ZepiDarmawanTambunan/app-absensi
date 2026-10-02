import 'dart:ui';

import 'package:camera/camera.dart';
import 'package:flutter/foundation.dart';
import 'package:google_mlkit_face_detection/google_mlkit_face_detection.dart';

/// Face found in a camera frame, with the signals the app needs.
class DetectedFace {
  final Rect boundingBox;

  /// 0..1 probability the eye is open (null when classification unavailable).
  final double? leftEyeOpenProbability;
  final double? rightEyeOpenProbability;

  /// Head rotation around the vertical axis, in degrees.
  final double? headEulerAngleY;

  const DetectedFace({
    required this.boundingBox,
    this.leftEyeOpenProbability,
    this.rightEyeOpenProbability,
    this.headEulerAngleY,
  });
}

/// On-device face detection via Google ML Kit.
///
/// Platform channels are required, so this class cannot run in `flutter test`
/// on the VM — it is exercised on a physical device / emulator instead.
class FaceDetectionService {
  final FaceDetector _detector;
  bool _busy = false;

  FaceDetectionService()
      : _detector = FaceDetector(
          options: FaceDetectorOptions(
            enableClassification: true,
            enableLandmarks: false,
            enableContours: false,
            enableTracking: false,
            minFaceSize: 0.15,
            performanceMode: FaceDetectorMode.fast,
          ),
        );

  /// Detects faces in [image]. Overlapping calls are dropped (empty list) to
  /// keep the preview stream real-time.
  Future<List<DetectedFace>> detect(
    CameraImage image, {
    required int sensorOrientation,
  }) async {
    if (_busy) return [];
    _busy = true;
    try {
      final inputImage = _toInputImage(image, sensorOrientation);
      if (inputImage == null) return [];
      final faces = await _detector.processImage(inputImage);
      return faces
          .map((f) => DetectedFace(
                boundingBox: f.boundingBox,
                leftEyeOpenProbability: f.leftEyeOpenProbability,
                rightEyeOpenProbability: f.rightEyeOpenProbability,
                headEulerAngleY: f.headEulerAngleY,
              ))
          .toList(growable: false);
    } finally {
      _busy = false;
    }
  }

  InputImage? _toInputImage(CameraImage image, int sensorOrientation) {
    final WriteBuffer allBytes = WriteBuffer();
    for (final Plane plane in image.planes) {
      allBytes.putUint8List(plane.bytes);
    }
    final Uint8List bytes = allBytes.done().buffer.asUint8List();

    final Size imageSize =
        Size(image.width.toDouble(), image.height.toDouble());
    final rotation =
        InputImageRotationValue.fromRawValue(sensorOrientation) ??
            InputImageRotation.rotation0deg;
    final format = InputImageFormatValue.fromRawValue(image.format.raw) ??
        InputImageFormat.nv21;
    if (image.planes.isEmpty) return null;

    final metadata = InputImageMetadata(
      size: imageSize,
      rotation: rotation,
      format: format,
      bytesPerRow: image.planes[0].bytesPerRow,
    );
    return InputImage.fromBytes(bytes: bytes, metadata: metadata);
  }

  Future<void> dispose() => _detector.close();
}
