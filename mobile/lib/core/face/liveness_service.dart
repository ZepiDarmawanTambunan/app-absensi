import 'package:camera/camera.dart';

import 'face_detection_service.dart';

/// Scores how likely the current frame shows a live person (0..1).
///
/// The score is sent as `liveness_score` to `POST /face/verify`; the server
/// enforces its threshold, audits rejections, and locks out repeat offenders.
/// A weak on-device score is therefore fail-closed server-side by design.
abstract class LivenessService {
  Future<double> scoreLiveness(CameraImage image, {DetectedFace? face});
}

/// Placeholder liveness score for development and automated tests.
///
/// TODO(M4): implement for real — track [DetectedFace.leftEyeOpenProbability]
/// / rightEyeOpenProbability across consecutive frames to detect a blink that
/// satisfies the server-issued challenge (`blink` / `turn_head`), and combine
/// with a texture-based passive check. Until then the server threshold still
/// applies, so spoof attempts are rejected server-side, not silently passed.
class StubLivenessService implements LivenessService {
  @override
  Future<double> scoreLiveness(CameraImage image, {DetectedFace? face}) async {
    return 0.92;
  }
}

LivenessService createLivenessService() => StubLivenessService();
