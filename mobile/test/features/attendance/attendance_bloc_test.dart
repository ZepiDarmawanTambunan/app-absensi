import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:app_absensi/core/models/attendance_log.dart';
import 'package:app_absensi/core/models/verify_result.dart';
import 'package:app_absensi/core/network/api_exception.dart';
import 'package:app_absensi/core/sync/pending_queue.dart';
import 'package:app_absensi/features/attendance/bloc/attendance_bloc.dart';
import 'package:app_absensi/features/attendance/domain/attendance_service.dart';

import '../../helpers/mocks.dart';

void main() {
  final log = AttendanceLog(
    id: 7,
    employeeId: 2,
    type: AttendanceType.checkIn,
    method: 'manual',
    verified: true,
    livenessScore: null,
    recordedAt: DateTime.utc(2026, 10, 2, 8),
  );
  final queued = PendingAction(
    id: 'q1',
    kind: 'check_in',
    employeeNo: 'EMP001',
    recordedAt: DateTime.utc(2026, 10, 2, 8, 5),
  );
  const rejected = VerifyResult(
    match: false,
    distance: 0,
    threshold: 0.5,
    livenessScore: 0.4,
    livenessThreshold: 0.7,
    livenessPassed: false,
    rejectReason: 'liveness_too_low',
  );

  group('AttendanceBloc', () {
    late MockAttendanceService service;

    setUp(() {
      service = MockAttendanceService();
    });

    blocTest<AttendanceBloc, AttendanceState>(
      'manual check-in records the log',
      setUp: () => when(() => service.checkInManual('EMP001'))
          .thenAnswer((_) async => AttendanceOutcome.recorded(log)),
      build: () => AttendanceBloc(service),
      act: (bloc) =>
          bloc.add(const AttendanceCheckInManualSubmitted('EMP001')),
      expect: () =>
          [const AttendanceInProgress(), AttendanceRecorded(log)],
    );

    blocTest<AttendanceBloc, AttendanceState>(
      'offline check-in is queued instead of failing',
      setUp: () => when(() => service.checkInManual('EMP001'))
          .thenAnswer((_) async => AttendanceOutcome.queued(queued)),
      build: () => AttendanceBloc(service),
      act: (bloc) =>
          bloc.add(const AttendanceCheckInManualSubmitted('EMP001')),
      expect: () =>
          [const AttendanceInProgress(), AttendanceQueued(queued)],
    );

    blocTest<AttendanceBloc, AttendanceState>(
      'face check-in surfaces a liveness rejection',
      setUp: () => when(() => service.checkInWithFace(
            employeeNo: 'EMP001',
            employeeId: 2,
            embedding: [0.1, 0.2],
            livenessScore: 0.4,
            challengeId: null,
          )).thenAnswer((_) async => AttendanceOutcome.faceRejected(rejected)),
      build: () => AttendanceBloc(service),
      act: (bloc) => bloc.add(const AttendanceCheckInFaceSubmitted(
        employeeNo: 'EMP001',
        employeeId: 2,
        embedding: [0.1, 0.2],
        livenessScore: 0.4,
      )),
      expect: () =>
          [const AttendanceInProgress(), const AttendanceFaceRejected(rejected)],
    );

    blocTest<AttendanceBloc, AttendanceState>(
      'server errors surface their message',
      setUp: () => when(() => service.checkInManual('EMP001')).thenThrow(
          const ApiException(code: 'conflict', message: 'Sudah check-in.')),
      build: () => AttendanceBloc(service),
      act: (bloc) =>
          bloc.add(const AttendanceCheckInManualSubmitted('EMP001')),
      expect: () =>
          [const AttendanceInProgress(), const AttendanceFailure('Sudah check-in.')],
    );

    blocTest<AttendanceBloc, AttendanceState>(
      'sync reports how many actions were pushed',
      setUp: () =>
          when(() => service.syncPending()).thenAnswer((_) async => 2),
      build: () => AttendanceBloc(service),
      act: (bloc) => bloc.add(const AttendancePendingSyncRequested()),
      expect: () =>
          [const AttendanceInProgress(), const AttendanceSynced(2)],
    );

    blocTest<AttendanceBloc, AttendanceState>(
      'pending count is reported',
      setUp: () => when(() => service.pendingActions())
          .thenAnswer((_) async => [queued]),
      build: () => AttendanceBloc(service),
      act: (bloc) => bloc.add(const AttendancePendingCountRequested()),
      expect: () => [const AttendancePendingCount(1)],
    );
  });
}
