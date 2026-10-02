import 'package:dio/dio.dart';

/// Typed error for every backend / network failure.
///
/// The backend answers errors as `{"error": {"code": "...", "message": "..."}}`;
/// this class unwraps that shape and also classifies transport failures so
/// callers can decide (e.g.) whether to queue work offline.
class ApiException implements Exception {
  /// Machine-readable code from the backend, or a client-side code such as
  /// `network`, `timeout`, `unauthorized`, `locked`, `unknown`.
  final String code;

  /// Human-readable message (safe to show in the UI after localisation).
  final String message;

  /// HTTP status when the failure came from a response, null otherwise.
  final int? statusCode;

  const ApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  /// True when there was no usable HTTP response (no connectivity, DNS, TLS…).
  /// Callers use this to decide whether to enqueue work in the offline queue.
  bool get isNetworkError => code == 'network' || code == 'timeout';

  bool get isUnauthorized => statusCode == 401;

  /// True for HTTP 423 — face verification locked after repeated failed
  /// liveness checks (anti-spoofing).
  bool get isLocked => statusCode == 423;

  factory ApiException.fromDio(DioException e) {
    final status = e.response?.statusCode;
    final data = e.response?.data;

    // Prefer the backend's own error envelope when present.
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return ApiException(
          code: err['code']?.toString() ?? 'unknown',
          message: err['message']?.toString() ?? 'Terjadi kesalahan.',
          statusCode: status,
        );
      }
    }

    switch (e.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.transformTimeout:
        return ApiException(
            code: 'timeout',
            message: 'Waktu permintaan habis. Periksa koneksi Anda.',
            statusCode: status);
      case DioExceptionType.connectionError:
      case DioExceptionType.badCertificate:
        return ApiException(
            code: 'network',
            message: 'Tidak dapat terhubung ke server.',
            statusCode: status);
      case DioExceptionType.cancel:
        return const ApiException(code: 'cancelled', message: 'Dibatalkan.');
      case DioExceptionType.badResponse:
        return ApiException(
          code: _codeFor(status),
          message: 'Server menjawab dengan status $status.',
          statusCode: status,
        );
      case DioExceptionType.unknown:
        return ApiException(
            code: 'network',
            message: e.message ?? 'Kesalahan jaringan.',
            statusCode: status);
    }
  }

  static String _codeFor(int? status) {
    if (status == null) return 'unknown';
    return switch (status) {
      400 => 'bad_request',
      401 => 'unauthorized',
      403 => 'forbidden',
      404 => 'not_found',
      409 => 'conflict',
      423 => 'locked',
      >= 500 => 'server_error',
      _ => 'unknown',
    };
  }

  @override
  String toString() => 'ApiException($code): $message';
}
