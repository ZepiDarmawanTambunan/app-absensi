import '../../../core/device/device_id_provider.dart';
import '../../../core/models/attendance_log.dart';
import '../../../core/models/verify_result.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/sync/pending_queue.dart';
import '../data/attendance_repository.dart';
import '../../face/data/face_repository.dart';

/// Outcome of one attendance attempt — exactly one field is set.
class AttendanceOutcome {
  final AttendanceLog? log;
  final PendingAction? queued;
  final VerifyResult? faceResult;

  const AttendanceOutcome._({this.log, this.queued, this.faceResult});

  factory AttendanceOutcome.recorded(AttendanceLog log) =>
      AttendanceOutcome._(log: log);

  factory AttendanceOutcome.queued(PendingAction action) =>
      AttendanceOutcome._(queued: action);

  factory AttendanceOutcome.faceRejected(VerifyResult result) =>
      AttendanceOutcome._(faceResult: result);

  bool get wasRecorded => log != null;
  bool get wasQueued => queued != null;
  bool get wasFaceRejected => faceResult != null;
}

/// Attendance use-cases: manual check-in/out, face check-in, offline sync.
///
/// Network failures during check-in/out degrade to the offline [PendingQueue]
/// instead of failing — attendance must not depend on perfect connectivity.
class AttendanceService {
  final AttendanceRepository _attendanceRepo;
  final FaceRepository _faceRepo;
  final PendingQueue _queue;
  final DeviceIdProvider _devices;

  AttendanceService({
    required AttendanceRepository attendanceRepository,
    required FaceRepository faceRepository,
    required PendingQueue pendingQueue,
    required DeviceIdProvider deviceIdProvider,
  })  : _attendanceRepo = attendanceRepository,
        _faceRepo = faceRepository,
        _queue = pendingQueue,
        _devices = deviceIdProvider;

  Future<AttendanceOutcome> checkInManual(String employeeNo) async {
    final deviceId = await _devices.getDeviceId();
    try {
      final log = await _attendanceRepo.checkIn(
        employeeNo: employeeNo,
        method: 'manual',
        deviceId: deviceId,
        platform: _devices.platformName,
      );
      return AttendanceOutcome.recorded(log);
    } on ApiException catch (e) {
      if (e.isNetworkError) return AttendanceOutcome.queued(await _enqueue('check_in', employeeNo));
      rethrow;
    }
  }

  /// Face check-in: liveness-gated 1:1 verification first, then check-in with
  /// `method: face`. A non-match returns [AttendanceOutcome.faceRejected];
  /// a 423 lock propagates as [ApiException].
  Future<AttendanceOutcome> checkInWithFace({
    required String employeeNo,
    required int employeeId,
    required List<double> embedding,
    required double livenessScore,
    String? challengeId,
  }) async {
    final deviceId = await _devices.getDeviceId();
    final verify = await _faceRepo.verify(
      employeeId: employeeId,
      embedding: embedding,
      livenessScore: livenessScore,
      deviceId: deviceId,
      challengeId: challengeId,
    );
    if (!verify.match) return AttendanceOutcome.faceRejected(verify);
    try {
      final log = await _attendanceRepo.checkIn(
        employeeNo: employeeNo,
        method: 'face',
        deviceId: deviceId,
        platform: _devices.platformName,
      );
      return AttendanceOutcome.recorded(log);
    } on ApiException catch (e) {
      if (e.isNetworkError) return AttendanceOutcome.queued(await _enqueue('check_in', employeeNo));
      rethrow;
    }
  }

  Future<AttendanceOutcome> checkOut(String employeeNo) async {
    final deviceId = await _devices.getDeviceId();
    try {
      final log = await _attendanceRepo.checkOut(
        employeeNo: employeeNo,
        deviceId: deviceId,
        platform: _devices.platformName,
      );
      return AttendanceOutcome.recorded(log);
    } on ApiException catch (e) {
      if (e.isNetworkError) return AttendanceOutcome.queued(await _enqueue('check_out', employeeNo));
      rethrow;
    }
  }

  /// Pushes queued actions to the server in FIFO order.
  /// Returns how many were synced. Stops at the first network failure;
  /// server-rejected actions (e.g. duplicates) are dropped so the queue drains.
  Future<int> syncPending() async {
    final items = await _queue.all();
    var synced = 0;
    for (final item in items) {
      try {
        final deviceId = await _devices.getDeviceId();
        if (item.kind == 'check_in') {
          await _attendanceRepo.checkIn(
            employeeNo: item.employeeNo,
            method: 'manual',
            deviceId: deviceId,
            platform: _devices.platformName,
          );
        } else {
          await _attendanceRepo.checkOut(
            employeeNo: item.employeeNo,
            deviceId: deviceId,
            platform: _devices.platformName,
          );
        }
        await _queue.remove(item.id);
        synced++;
      } on ApiException catch (e) {
        if (!e.isNetworkError) await _queue.remove(item.id);
        break;
      }
    }
    return synced;
  }

  Future<List<PendingAction>> pendingActions() => _queue.all();

  Future<PendingAction> _enqueue(String kind, String employeeNo) async {
    final action = PendingAction(
      id: DateTime.now().microsecondsSinceEpoch.toString(),
      kind: kind,
      employeeNo: employeeNo,
      recordedAt: DateTime.now(),
    );
    await _queue.enqueue(action);
    return action;
  }
}
