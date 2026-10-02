import 'api_exception.dart';

/// Unwraps the backend JSON envelope.
///
/// Success: `{"data": ...}` — Error: `{"error": {"code": "...", "message": "..."}}`.
class ApiEnvelope {
  /// Extracts `data` and decodes it with [decode].
  /// Throws [ApiException] when the payload carries `error` or is malformed.
  static T data<T>(Map<String, dynamic> json, T Function(Object? raw) decode) {
    if (json.containsKey('data')) {
      return decode(json['data']);
    }
    final err = json['error'];
    if (err is Map<String, dynamic>) {
      throw ApiException(
        code: err['code']?.toString() ?? 'unknown',
        message: err['message']?.toString() ?? 'Terjadi kesalahan.',
      );
    }
    throw const ApiException(
        code: 'malformed', message: 'Format respons server tidak dikenal.');
  }

  /// Variant of [data] for endpoints returning a JSON array in `data`.
  static List<T> dataList<T>(
      Map<String, dynamic> json, T Function(Map<String, dynamic> raw) fromJson) {
    return data(json, (raw) {
      final list = raw as List<dynamic>;
      return list
          .map((e) => fromJson(e as Map<String, dynamic>))
          .toList(growable: false);
    });
  }
}
