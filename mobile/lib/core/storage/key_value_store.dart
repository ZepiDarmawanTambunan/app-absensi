import 'package:shared_preferences/shared_preferences.dart';

/// Tiny key-value abstraction over plain (non-secret) local storage.
///
/// Used for the offline attendance queue, the biometric-consent flag, etc.
/// Secrets (tokens) must go through [SecureTokenStorage] instead.
abstract class KeyValueStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
  Future<void> remove(String key);
}

/// Production implementation backed by SharedPreferences.
class SharedPreferencesStore implements KeyValueStore {
  final SharedPreferences _prefs;

  SharedPreferencesStore(this._prefs);

  @override
  Future<String?> read(String key) async => _prefs.getString(key);

  @override
  Future<void> write(String key, String value) async {
    await _prefs.setString(key, value);
  }

  @override
  Future<void> remove(String key) async {
    await _prefs.remove(key);
  }
}

/// In-memory implementation for unit/widget tests (no plugins needed).
class InMemoryStore implements KeyValueStore {
  final Map<String, String> _map = {};

  @override
  Future<String?> read(String key) async => _map[key];

  @override
  Future<void> write(String key, String value) async {
    _map[key] = value;
  }

  @override
  Future<void> remove(String key) async {
    _map.remove(key);
  }
}
