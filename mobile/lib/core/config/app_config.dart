/// Runtime configuration for the app.
///
/// Values come from `--dart-define` so each flavor (dev/staging/prod) can
/// point at a different backend without code changes:
///
/// ```bash
/// flutter run --dart-define API_BASE_URL=https://api.kantor.id/api/v1
/// ```
///
/// NOTE: `10.0.2.2` is the Android emulator's alias for the host machine's
/// localhost. Use your machine's LAN IP when testing on a physical device.
class AppConfig {
  /// Base URL of the backend REST API, e.g. `https://api.kantor.id/api/v1`.
  final String baseUrl;

  /// When true, face embeddings are extracted with the bundled TFLite
  /// MobileFaceNet model; when false (default) a deterministic stub is used.
  /// See [FaceEmbeddingService] docs and mobile/README.md.
  final bool useTfliteEmbedding;

  const AppConfig({
    required this.baseUrl,
    required this.useTfliteEmbedding,
  });

  factory AppConfig.fromEnvironment() => const AppConfig(
        baseUrl: String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://10.0.2.2:8080/api/v1',
        ),
        useTfliteEmbedding: bool.fromEnvironment(
          'USE_TFLITE_EMBEDDING',
          defaultValue: false,
        ),
      );
}
