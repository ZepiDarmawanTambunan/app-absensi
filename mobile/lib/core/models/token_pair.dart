import 'package:equatable/equatable.dart';

import '../network/envelope.dart';

/// JWT pair issued by `POST /api/v1/auth/login` (and `/auth/refresh`).
class TokenPair extends Equatable {
  final String accessToken;
  final String refreshToken;

  /// Seconds until the access token expires (informational; the app checks
  /// expiry from the JWT `exp` claim instead).
  final int expiresIn;

  const TokenPair({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresIn,
  });

  factory TokenPair.fromEnvelope(Map<String, dynamic> json) =>
      ApiEnvelope.data(json, (raw) {
        final m = raw as Map<String, dynamic>;
        return TokenPair(
          accessToken: m['access_token'] as String,
          refreshToken: m['refresh_token'] as String,
          expiresIn: (m['expires_in'] as num?)?.toInt() ?? 0,
        );
      });

  @override
  List<Object?> get props => [accessToken, refreshToken, expiresIn];
}
