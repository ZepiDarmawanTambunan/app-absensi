import '../../../core/models/attendance_log.dart';
import '../../../core/network/api_client.dart';
import '../../../core/network/envelope.dart';

/// Raw attendance endpoints. Orchestration (face gate, offline queue) lives
/// in [AttendanceService]; UI state in [AttendanceBloc].
class AttendanceRepository {
  final ApiClient _api;

  AttendanceRepository(this._api);

  Future<AttendanceLog> checkIn({
    required String employeeNo,
    required String method,
    String? deviceId,
    String? platform,
  }) async {
    final json = await _api.post('/attendance/check-in', data: {
      'employee_no': employeeNo,
      'method': method,
      if (deviceId != null) 'device_id': deviceId,
      if (platform != null) 'platform': platform,
    });
    return ApiEnvelope.data(
        json, (raw) => AttendanceLog.fromJson(raw as Map<String, dynamic>));
  }

  Future<AttendanceLog> checkOut({
    required String employeeNo,
    String? deviceId,
    String? platform,
  }) async {
    final json = await _api.post('/attendance/check-out', data: {
      'employee_no': employeeNo,
      if (deviceId != null) 'device_id': deviceId,
      if (platform != null) 'platform': platform,
    });
    return ApiEnvelope.data(
        json, (raw) => AttendanceLog.fromJson(raw as Map<String, dynamic>));
  }

  Future<List<AttendanceLog>> history({
    required int employeeId,
    DateTime? from,
    DateTime? to,
    int limit = 50,
  }) async {
    final json = await _api.get('/attendance', query: {
      'employee_id': employeeId,
      if (from != null) 'from': from.toUtc().toIso8601String(),
      if (to != null) 'to': to.toUtc().toIso8601String(),
      'limit': limit,
    });
    return ApiEnvelope.dataList(json, AttendanceLog.fromJson);
  }
}
