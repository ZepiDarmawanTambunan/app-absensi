import 'package:equatable/equatable.dart';

/// One captured frame, returned by [FaceCapturePage] to its caller.
class FaceCaptureResult extends Equatable {
  final List<double> embedding;

  /// 0..1 capture-time quality heuristic (see [FaceCapturePage]).
  final double quality;
  final double livenessScore;
  final String? challengeId;

  const FaceCaptureResult({
    required this.embedding,
    required this.quality,
    required this.livenessScore,
    this.challengeId,
  });

  @override
  List<Object?> get props => [embedding, quality, livenessScore, challengeId];
}
