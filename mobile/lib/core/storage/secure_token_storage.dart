import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Stores auth tokens in the platform keychain/keystore.
///
/// NEVER use SharedPreferences (or any plain storage) for these values.
class SecureTokenStorage {
  static const _accessKey = 'auth.access_token';
  static const _refreshKey = 'auth.refresh_token';

  final FlutterSecureStorage _storage;

  SecureTokenStorage(this._storage);

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await _storage.write(key: _accessKey, value: accessToken);
    await _storage.write(key: _refreshKey, value: refreshToken);
  }

  Future<String?> get accessToken => _storage.read(key: _accessKey);

  Future<String?> get refreshToken => _storage.read(key: _refreshKey);

  Future<void> clear() => _storage.deleteAll();
}
