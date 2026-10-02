import 'package:equatable/equatable.dart';

import '../network/envelope.dart';

/// Result of `POST /api/v1/face/verify` (1:1 against the claimed employee).
class VerifyResult extends Equatable {
  final bool match;

  /// Cosine distance probe-vs-template (0 when liveness failed — the
  /// embedding is not compared in that case).
  final double distance;
  final double threshold;

  final double? livenessScore;
  final double? livenessThreshold;
  final bool? livenessPassed;

  /// e.g. `liveness_too_low` — present when the liveness gate rejected.
  final String? rejectReason;

  /// True when the response was HTTP 423 (handled via [ApiException.isLocked]
  /// instead); kept for completeness.
  final bool locked;
  final int? retryAfter;

  const VerifyResult({
    required this.match,
    required this.distance,
    required this.threshold,
    this.livenessScore,
    this.livenessThreshold,
    this.livenessPassed,
    this.rejectReason,
    this.locked = false,
    this.retryAfter,
  });

  factory VerifyResult.fromEnvelope(Map<String, dynamic> json) =>
      ApiEnvelope.data(json, (raw) {
        final m = raw as Map<String, dynamic>;
        final liv = m['liveness'];
        final livMap = liv is Map<String, dynamic> ? liv : null;
        return VerifyResult(
          match: m['match'] as bool? ?? false,
          distance: (m['distance'] as num?)?.toDouble() ?? 0,
          threshold: (m['threshold'] as num?)?.toDouble() ?? 0.5,
          livenessScore: (livMap?['score'] as num?)?.toDouble(),
          livenessThreshold:
              (livMap?['threshold'] as num?)?.toDouble(),
          livenessPassed: livMap?['passed'] as bool?,
          rejectReason: m['reject_reason']?.toString(),
        );
      });

  @override
  List<Object?> get props => [
        match,
        distance,
        threshold,
        livenessScore,
        livenessThreshold,
        livenessPassed,
        rejectReason,
        locked,
        retryAfter,
      ];
}
