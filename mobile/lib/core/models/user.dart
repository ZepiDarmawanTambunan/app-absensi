import 'package:equatable/equatable.dart';

import '../utils/jwt.dart';

/// The logged-in operator, read from the JWT payload (display only).
class AppUser extends Equatable {
  final int id;
  final String name;
  final String email;

  const AppUser({required this.id, required this.name, required this.email});

  /// The backend does not expose a `/me` endpoint, so the operator identity
  /// is taken from the access token's claims. Auth trust stays server-side.
  factory AppUser.fromJwt(String token) {
    final claims = JwtDecoder.decodePayload(token);
    return AppUser(
      id: (claims['user_id'] as num?)?.toInt() ?? 0,
      name: claims['name']?.toString() ?? 'Operator',
      email: claims['email']?.toString() ?? '',
    );
  }

  @override
  List<Object?> get props => [id, name, email];
}
