import 'dart:convert';

/// Minimal JWT payload reader — signature is NOT verified.
///
/// Used only to display the logged-in operator's identity and to check token
/// expiry client-side. Trust decisions (auth) always happen on the server.
class JwtDecoder {
  /// Returns the decoded payload claims, or an empty map when undecodable.
  static Map<String, dynamic> decodePayload(String token) {
    try {
      final parts = token.split('.');
      if (parts.length != 3) return {};
      final normalized = base64Url.normalize(parts[1]);
      final json = utf8.decode(base64Url.decode(normalized));
      final map = jsonDecode(json);
      if (map is Map<String, dynamic>) return map;
      return {};
    } catch (_) {
      return {};
    }
  }

  /// Expiry instant from the `exp` claim, or null when absent/undecodable.
  static DateTime? expiryOf(String token) {
    final exp = decodePayload(token)['exp'];
    if (exp is num) {
      return DateTime.fromMillisecondsSinceEpoch(exp.toInt() * 1000);
    }
    return null;
  }

  /// True when the token is missing an `exp` claim or it lies in the past
  /// (minus a small clock-skew allowance).
  static bool isExpired(String token, {Duration skew = const Duration(seconds: 30)}) {
    final exp = expiryOf(token);
    if (exp == null) return true;
    return exp.subtract(skew).isBefore(DateTime.now());
  }
}
