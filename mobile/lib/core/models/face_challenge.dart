import 'package:equatable/equatable.dart';

import '../network/envelope.dart';

/// Single-use liveness challenge from `GET /api/v1/face/challenge`.
class FaceChallenge extends Equatable {
  final String challengeId;

  /// `blink` or `turn_head` — the action the user must perform on-device.
  final String type;

  /// Seconds until the challenge expires (server-side, ~120s).
  final int expiresIn;

  const FaceChallenge({
    required this.challengeId,
    required this.type,
    required this.expiresIn,
  });

  factory FaceChallenge.fromEnvelope(Map<String, dynamic> json) =>
      ApiEnvelope.data(json, (raw) {
        final m = raw as Map<String, dynamic>;
        return FaceChallenge(
          challengeId: m['challenge_id'] as String,
          type: m['type'] as String,
          expiresIn: (m['expires_in'] as num?)?.toInt() ?? 120,
        );
      });

  /// Human instruction shown on the capture screen.
  String get instruction => switch (type) {
        'blink' => 'Kedipkan mata Anda',
        'turn_head' => 'Tengokkan kepala ke kiri dan kanan',
        _ => 'Ikuti instruksi di layar',
      };

  @override
  List<Object?> get props => [challengeId, type, expiresIn];
}
