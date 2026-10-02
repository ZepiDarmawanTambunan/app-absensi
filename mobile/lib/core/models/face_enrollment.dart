import 'package:equatable/equatable.dart';

import '../network/envelope.dart';

/// Stored face template from `POST /api/v1/face/enroll`.
///
/// The embedding itself is never exposed to the client (`json:"-"` on the
/// server) — only metadata comes back.
class FaceEnrollment extends Equatable {
  final int id;
  final int employeeId;
  final int dimension;
  final double qualityScore;
  final DateTime enrolledAt;

  const FaceEnrollment({
    required this.id,
    required this.employeeId,
    required this.dimension,
    required this.qualityScore,
    required this.enrolledAt,
  });

  factory FaceEnrollment.fromEnvelope(Map<String, dynamic> json) =>
      ApiEnvelope.data(json, (raw) {
        final m = raw as Map<String, dynamic>;
        return FaceEnrollment(
          id: (m['id'] as num).toInt(),
          employeeId: (m['employee_id'] as num).toInt(),
          dimension: (m['dimension'] as num?)?.toInt() ?? 0,
          qualityScore: (m['quality_score'] as num?)?.toDouble() ?? 0,
          enrolledAt: DateTime.parse(m['enrolled_at'] as String),
        );
      });

  @override
  List<Object?> get props =>
      [id, employeeId, dimension, qualityScore, enrolledAt];
}
