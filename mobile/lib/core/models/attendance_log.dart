import 'package:equatable/equatable.dart';

/// Wire values of an attendance record's `type` column.
enum AttendanceType { checkIn, checkOut }

AttendanceType attendanceTypeFromString(String value) =>
    switch (value) {
      'check_in' => AttendanceType.checkIn,
      'check_out' => AttendanceType.checkOut,
      _ => throw ArgumentError('Unknown attendance type: $value'),
    };

String attendanceTypeToString(AttendanceType type) => switch (type) {
      AttendanceType.checkIn => 'check_in',
      AttendanceType.checkOut => 'check_out',
    };

/// One row of `attendance_logs`.
class AttendanceLog extends Equatable {
  final int id;
  final int employeeId;
  final AttendanceType type;

  /// `face` or `manual`.
  final String method;
  final bool verified;
  final double? livenessScore;
  final DateTime recordedAt;

  const AttendanceLog({
    required this.id,
    required this.employeeId,
    required this.type,
    required this.method,
    required this.verified,
    required this.livenessScore,
    required this.recordedAt,
  });

  factory AttendanceLog.fromJson(Map<String, dynamic> json) => AttendanceLog(
        id: (json['id'] as num).toInt(),
        employeeId: (json['employee_id'] as num).toInt(),
        type: attendanceTypeFromString(json['type'] as String),
        method: json['method'] as String? ?? 'manual',
        verified: json['verified'] as bool? ?? false,
        livenessScore: (json['liveness_score'] as num?)?.toDouble(),
        recordedAt: DateTime.parse(json['recorded_at'] as String),
      );

  @override
  List<Object?> get props =>
      [id, employeeId, type, method, verified, livenessScore, recordedAt];
}
