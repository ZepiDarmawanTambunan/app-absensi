import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../../core/models/attendance_log.dart';
import '../../../core/models/verify_result.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/sync/pending_queue.dart';
import '../domain/attendance_service.dart';

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

sealed class AttendanceEvent extends Equatable {
  const AttendanceEvent();

  @override
  List<Object?> get props => [];
}

final class AttendanceCheckInManualSubmitted extends AttendanceEvent {
  final String employeeNo;

  const AttendanceCheckInManualSubmitted(this.employeeNo);

  @override
  List<Object?> get props => [employeeNo];
}

final class AttendanceCheckInFaceSubmitted extends AttendanceEvent {
  final String employeeNo;
  final int employeeId;
  final List<double> embedding;
  final double livenessScore;
  final String? challengeId;

  const AttendanceCheckInFaceSubmitted({
    required this.employeeNo,
    required this.employeeId,
    required this.embedding,
    required this.livenessScore,
    this.challengeId,
  });

  @override
  List<Object?> get props =>
      [employeeNo, employeeId, embedding, livenessScore, challengeId];
}

final class AttendanceCheckOutSubmitted extends AttendanceEvent {
  final String employeeNo;

  const AttendanceCheckOutSubmitted(this.employeeNo);

  @override
  List<Object?> get props => [employeeNo];
}

final class AttendancePendingSyncRequested extends AttendanceEvent {
  const AttendancePendingSyncRequested();
}

final class AttendancePendingCountRequested extends AttendanceEvent {
  const AttendancePendingCountRequested();
}

// ---------------------------------------------------------------------------
// States
// ---------------------------------------------------------------------------

sealed class AttendanceState extends Equatable {
  const AttendanceState();

  @override
  List<Object?> get props => [];
}

final class AttendanceInitial extends AttendanceState {
  const AttendanceInitial();
}

final class AttendanceInProgress extends AttendanceState {
  const AttendanceInProgress();
}

final class AttendanceRecorded extends AttendanceState {
  final AttendanceLog log;

  const AttendanceRecorded(this.log);

  @override
  List<Object?> get props => [log];
}

final class AttendanceQueued extends AttendanceState {
  final PendingAction action;

  const AttendanceQueued(this.action);

  @override
  List<Object?> get props => [action];
}

final class AttendanceFaceRejected extends AttendanceState {
  final VerifyResult result;

  const AttendanceFaceRejected(this.result);

  @override
  List<Object?> get props => [result];
}

final class AttendanceSynced extends AttendanceState {
  final int count;

  const AttendanceSynced(this.count);

  @override
  List<Object?> get props => [count];
}

final class AttendancePendingCount extends AttendanceState {
  final int count;

  const AttendancePendingCount(this.count);

  @override
  List<Object?> get props => [count];
}

final class AttendanceFailure extends AttendanceState {
  final String message;

  const AttendanceFailure(this.message);

  @override
  List<Object?> get props => [message];
}

// ---------------------------------------------------------------------------
// Bloc
// ---------------------------------------------------------------------------

class AttendanceBloc extends Bloc<AttendanceEvent, AttendanceState> {
  final AttendanceService _service;

  AttendanceBloc(this._service) : super(const AttendanceInitial()) {
    on<AttendanceCheckInManualSubmitted>(_onManualCheckIn);
    on<AttendanceCheckInFaceSubmitted>(_onFaceCheckIn);
    on<AttendanceCheckOutSubmitted>(_onCheckOut);
    on<AttendancePendingSyncRequested>(_onSync);
    on<AttendancePendingCountRequested>(_onCount);
  }

  Future<void> _onManualCheckIn(AttendanceCheckInManualSubmitted event,
      Emitter<AttendanceState> emit) async {
    emit(const AttendanceInProgress());
    await _guard(emit, () => _service.checkInManual(event.employeeNo));
  }

  Future<void> _onFaceCheckIn(AttendanceCheckInFaceSubmitted event,
      Emitter<AttendanceState> emit) async {
    emit(const AttendanceInProgress());
    await _guard(
        emit,
        () => _service.checkInWithFace(
              employeeNo: event.employeeNo,
              employeeId: event.employeeId,
              embedding: event.embedding,
              livenessScore: event.livenessScore,
              challengeId: event.challengeId,
            ));
  }

  Future<void> _onCheckOut(AttendanceCheckOutSubmitted event,
      Emitter<AttendanceState> emit) async {
    emit(const AttendanceInProgress());
    await _guard(emit, () => _service.checkOut(event.employeeNo));
  }

  Future<void> _onSync(AttendancePendingSyncRequested event,
      Emitter<AttendanceState> emit) async {
    emit(const AttendanceInProgress());
    try {
      final count = await _service.syncPending();
      emit(AttendanceSynced(count));
    } on ApiException catch (e) {
      emit(AttendanceFailure(e.message));
    } catch (_) {
      emit(const AttendanceFailure('Gagal menyinkronkan antrean.'));
    }
  }

  Future<void> _onCount(AttendancePendingCountRequested event,
      Emitter<AttendanceState> emit) async {
    final count = (await _service.pendingActions()).length;
    emit(AttendancePendingCount(count));
  }

  Future<void> _guard(Emitter<AttendanceState> emit,
      Future<AttendanceOutcome> Function() run) async {
    try {
      final outcome = await run();
      if (outcome.wasRecorded) {
        emit(AttendanceRecorded(outcome.log!));
      } else if (outcome.wasQueued) {
        emit(AttendanceQueued(outcome.queued!));
      } else if (outcome.wasFaceRejected) {
        emit(AttendanceFaceRejected(outcome.faceResult!));
      }
    } on ApiException catch (e) {
      emit(AttendanceFailure(e.message));
    } catch (_) {
      emit(const AttendanceFailure('Terjadi kesalahan tak terduga.'));
    }
  }
}
