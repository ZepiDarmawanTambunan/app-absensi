import 'package:mocktail/mocktail.dart';

import 'package:app_absensi/features/attendance/domain/attendance_service.dart';
import 'package:app_absensi/features/auth/domain/auth_service.dart';
import 'package:app_absensi/features/enrollment/domain/enrollment_service.dart';
import 'package:app_absensi/features/history/domain/history_service.dart';

class MockAuthService extends Mock implements AuthService {}

class MockAttendanceService extends Mock implements AttendanceService {}

class MockEnrollmentService extends Mock implements EnrollmentService {}

class MockHistoryService extends Mock implements HistoryService {}
