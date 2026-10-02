import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:app_absensi/core/face/face_embedding_service.dart';
import 'package:app_absensi/core/models/face_enrollment.dart';
import 'package:app_absensi/features/enrollment/bloc/enrollment_bloc.dart';
import 'package:app_absensi/features/enrollment/domain/enrollment_service.dart';

import '../../helpers/mocks.dart';

void main() {
  const frames = [
    FaceFrame(embedding: [0.1, 0.2], quality: 0.9),
    FaceFrame(embedding: [0.2, 0.1], quality: 0.8),
  ];
  final enrollment = FaceEnrollment(
    id: 3,
    employeeId: 2,
    dimension: 128,
    qualityScore: 0.85,
    enrolledAt: DateTime.utc(2026, 10, 2, 9),
  );

  group('EnrollmentBloc', () {
    late MockEnrollmentService service;

    setUp(() {
      service = MockEnrollmentService();
    });

    blocTest<EnrollmentBloc, EnrollmentState>(
      'stores the template on success',
      setUp: () => when(() =>
              service.enrollFromFrames(employeeId: 2, frames: frames))
          .thenAnswer((_) async => enrollment),
      build: () => EnrollmentBloc(service),
      act: (bloc) =>
          bloc.add(const EnrollmentSubmitted(employeeId: 2, frames: frames)),
      expect: () =>
          [const EnrollmentInProgress(), EnrollmentSuccess(enrollment)],
    );

    blocTest<EnrollmentBloc, EnrollmentState>(
      'surfaces quality-gate failures',
      setUp: () => when(() =>
              service.enrollFromFrames(employeeId: 2, frames: frames))
          .thenThrow(const EnrollmentException('Kualitas kurang.')),
      build: () => EnrollmentBloc(service),
      act: (bloc) =>
          bloc.add(const EnrollmentSubmitted(employeeId: 2, frames: frames)),
      expect: () =>
          [const EnrollmentInProgress(), const EnrollmentFailure('Kualitas kurang.')],
    );
  });
}
