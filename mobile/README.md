# app-absensi — Mobile (Flutter)

Aplikasi mobile **app-absensi**: absensi pegawai dengan face recognition untuk
Android & iOS. Berpasangan dengan `backend/` (REST API Go + MySQL) di monorepo
ini.

## Fitur (Milestone 4)

- **Login operator** (JWT, auto-refresh token, sesi dipulihkan otomatis)
- **Check-in / check-out manual** dengan nomor pegawai
- **Check-in dengan wajah**: kamera → deteksi wajah on-device (ML Kit) →
  embedding on-device → verifikasi 1:1 ke server + liveness challenge
  (`blink` / `turn_head`)
- **Enroll wajah**: ambil 3 frame → simpan template ke server
- **Riwayat kehadiran** per pegawai (data server + antrean offline)
- **Mode offline**: check-in/out yang gagal karena jaringan masuk antrean
  lokal dan disinkronkan saat online kembali (badge *pending* di UI)
- **Persetujuan biometrik** sebelum pengambilan wajah pertama; foto mentah
  tidak pernah dikirim/disimpan — hanya embedding

## Prasyarat

- Flutter SDK 3.47+ (`flutter --version`)
- Backend berjalan dan terjangkau dari perangkat
  (`backend/README.md` — bagian instalasi)
- Perangkat fisik / emulator berkamera untuk fitur wajah

## Instalasi & menjalankan

```bash
cd mobile
flutter pub get

# Android emulator (10.0.2.2 = localhost mesin host):
flutter run --dart-define API_BASE_URL=http://10.0.2.2:8080/api/v1

# Perangkat fisik (ganti dengan IP LAN mesin backend):
flutter run --dart-define API_BASE_URL=http://192.168.1.10:8080/api/v1

# Dengan model embedding asli (lihat bawah):
flutter run \
  --dart-define API_BASE_URL=http://10.0.2.2:8080/api/v1 \
  --dart-define USE_TFLITE_EMBEDDING=true
```

| `--dart-define`         | Default                         | Keterangan                              |
|-------------------------|---------------------------------|-----------------------------------------|
| `API_BASE_URL`          | `http://10.0.2.2:8080/api/v1`   | Base URL backend REST API               |
| `USE_TFLITE_EMBEDDING`  | `false`                         | `true` = pakai model MobileFaceNet      |

## Arsitektur lapisan

Satu pola di seluruh fitur — **Widget → Bloc → Service → Repository → ApiClient**,
dengan `flutter_bloc` (Bloc, bukan Cubit) sebagai satu-satunya state management:

```
lib/
  main.dart                 # composition root (rakit semua dependency)
  app.dart                  # MaterialApp + MultiBlocProvider + AuthGate
  home_page.dart            # bottom navigation
  core/
    config/                 # AppConfig (--dart-define)
    network/                # ApiClient (Dio + auto refresh token), ApiException, envelope
    models/                 # User, TokenPair, Employee, AttendanceLog, VerifyResult, ...
    storage/                # SecureTokenStorage (keychain), KeyValueStore
    sync/                   # PendingQueue (antrean offline)
    face/                   # FaceDetectionService, FaceEmbeddingService, LivenessService
    device/                 # DeviceIdProvider
    permissions/            # CameraPermission
    widgets/                # AppButton, AppTextField, ErrorView
    utils/                  # JwtDecoder
  features/
    auth/                   # data/auth_repository → domain/auth_service → bloc/auth_bloc → presentation/login_page
    attendance/             # + presentation/face_capture_page.dart & widgets/face_camera_view.dart
    enrollment/
    history/
    face/                   # data/face_repository.dart (API wajah, dipakai enrollment & attendance)
```

Aturan: semua HTTP lewat `ApiClient` (tidak ada Dio/`http` di widget);
token hanya di `flutter_secure_storage`; jangan pernah log token/embedding/foto.

## Pipeline wajah on-device

1. `FaceDetectionService` — deteksi wajah via `google_mlkit_face_detection`
   (butuh perangkat fisik).
2. `FaceEmbeddingService` — ekstraksi embedding:
   - **Default: `StubEmbeddingService`** — vektor pseudo acak yang deterministik
     dari frame (untuk development & test otomatis; BUKAN biometrik asli).
   - **Asli: `TfliteFaceEmbeddingService`** — model MobileFaceNet
     (output 192-d, ternormalisasi L2).
3. `LivenessService` — skor liveness; saat ini **stub (0,92)**. TODO: deteksi
   kedipan via probabilitas mata ML Kit antar-frame.

### Model MobileFaceNet

Model tidak dibundel (lisensi/ukuran). Untuk memakai yang asli:

1. Unduh `mobilefacenet.tflite` (input 112×112×3, output 192).
2. Taruh di `mobile/assets/models/mobilefacenet.tflite`
   (folder sudah didaftarkan di `pubspec.yaml`).
3. Jalankan dengan `--dart-define USE_TFLITE_EMBEDDING=true`.
4. Validasi terhadap vektor uji resmi MobileFaceNet sebelum produksi.

## Yang bisa vs tidak bisa dites di VM

| Bisa di VM (`flutter test` / `flutter analyze`) | Butuh perangkat fisik / emulator |
|---|---|
| Semua Bloc (`bloc_test`), service, repository (mock) | `FaceCameraView` (kamera) |
| `PendingQueue`, `ApiException`, envelope | `FaceDetectionService` (ML Kit) |
| Widget test `LoginPage` | Ekstraksi embedding TFLite asli |
| `StubEmbeddingService` / `StubLivenessService` | Izin kamera & alur consent end-to-end |

## Testing

```bash
flutter analyze   # harus bersih
flutter test      # unit + bloc + widget test
```

Build APK/AAB sengaja tidak dilakukan di VM (butuh Android SDK);
dilakukan di mesin developer atau CI.
