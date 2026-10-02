import 'package:permission_handler/permission_handler.dart';

/// Camera permission gate — called before any capture screen opens.
class CameraPermission {
  /// Returns true when camera access is granted (requesting it if needed).
  Future<bool> ensureGranted() async {
    final current = await Permission.camera.status;
    if (current.isGranted) return true;
    final next = await Permission.camera.request();
    return next.isGranted;
  }
}
