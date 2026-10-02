import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';

/// Provides a stable per-device identifier for the `device_id` API field.
///
/// Failures degrade to `'unknown-device'` — device identification must never
/// break an attendance attempt.
class DeviceIdProvider {
  final DeviceInfoPlugin _plugin;

  DeviceIdProvider([DeviceInfoPlugin? plugin])
      : _plugin = plugin ?? DeviceInfoPlugin();

  Future<String> getDeviceId() async {
    try {
      if (Platform.isAndroid) {
        final info = await _plugin.androidInfo;
        return info.id.isEmpty ? 'android-unknown' : info.id;
      }
      if (Platform.isIOS) {
        final info = await _plugin.iosInfo;
        return info.identifierForVendor ?? 'ios-unknown';
      }
    } catch (_) {
      return 'unknown-device';
    }
    return 'unknown-device';
  }

  String get platformName {
    if (Platform.isAndroid) return 'android';
    if (Platform.isIOS) return 'ios';
    return 'other';
  }
}
