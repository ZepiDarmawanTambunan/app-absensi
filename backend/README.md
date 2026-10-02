# app-absensi — Backend

Backend REST API untuk **app-absensi**, aplikasi absensi pegawai dengan **face recognition** (Android & iOS).

Aplikasi ini ditujukan untuk perusahaan/kantor yang membutuhkan pencatatan kehadiran akurat dan sulit dimanipulasi: karyawan absen dengan wajahnya sendiri lewat aplikasi mobile, sistem memverifikasi identitas sekaligus memastikan wajah tersebut **asli dan hidup** (bukan foto atau video).

## Arsitektur

Repo ini adalah **monorepo**:

```
app-absensi/
├── backend/   # ← Anda di sini: REST API Go + MySQL
└── mobile/    # Aplikasi Flutter (Android & iOS) — menyusul di Milestone 4
```

Prinsip arsitektur face recognition: **on-device**. Aplikasi mobile yang mendeteksi wajah dan mengekstrak *embedding*; server hanya menyimpan template dan membandingkan. **Foto wajah mentah tidak pernah dikirim ke atau disimpan di server** — yang tersimpan hanya vektor embedding.

## Fitur per Milestone

### Milestone 1 — API absensi ✅
- Autentikasi operator: register, login, refresh token (**JWT**, access 15 menit + refresh 7 hari)
- CRUD data karyawan (soft-delete)
- Check-in / check-out dengan aturan bisnis: tolak check-in ganda (409), tolak check-out tanpa check-in terbuka (404)
- Riwayat kehadiran dengan filter tanggal
- Migrasi database berversi (users, employees, devices, attendance_logs)

### Milestone 2 — Enroll & verifikasi wajah ✅
- `POST /face/enroll`: simpan template wajah pegawai dari 1–5 frame (yang lolos quality ≥ 0.5 dirata-rata menjadi 1 template ternormalisasi)
- `POST /face/verify`: verifikasi **1:1** — embedding dibandingkan hanya dengan template milik `employee_id` yang diklaim, tidak pernah 1:N ke seluruh database
- Dimensi embedding yang didukung: 128 / 192 / 512; threshold cosine distance dapat di-tune via config (default 0.50)

### Milestone 3 — Anti-spoofing ✅
- **Liveness enforcement**: setiap verify wajib menyertakan `liveness_score` dari model liveness di perangkat. Skor di bawah threshold (default **0.70**) → `match=false` dengan `reject_reason: liveness_too_low`, **tanpa** membandingkan embedding
- **Audit**: setiap penolakan liveness dicatat ke tabel `spoof_attempts` (skor, alasan, device_id, waktu)
- **Lockout otomatis**: setelah **5** kegagalan liveness dalam 1 jam, verifikasi wajah dikunci selama **15 menit** → API menjawab `423 Locked` beserta `retry_after` (detik). Kunci lepas sendiri setelah waktunya habis; kegagalan lama (>1 jam) tidak dihitung
- **Challenge-response**: `GET /face/challenge` menerbitkan tantangan sekali pakai (`blink` / `turn_head`, kedaluwarsa 2 menit). Aplikasi mobile melaksanakannya dan menyertakan `challenge_id` saat verify; id yang tidak dikenal, kedaluwarsa, atau dipakai ulang ditolak (anti-replay)

### Rencana berikutnya
- **Milestone 4** — aplikasi mobile Flutter (kamera, ML Kit, antrean offline)
- **Milestone 5** — testing menyeluruh
- **Milestone 6** — dokumentasi akhir

## Instalasi dari Nol

### Prasyarat
- Go 1.22+ (`go version`)
- MySQL 8 (server berjalan lokal atau terjangkau)
- Git
- (Opsional) `migrate` CLI dari [golang-migrate](https://github.com/golang-migrate/migrate) untuk menjalankan migrasi

### 1. Clone & masuk ke direktori backend

```bash
git clone https://github.com/ZepiDarmawanTambunan/app-absensi.git
cd app-absensi/backend
```

### 2. Siapkan database

```sql
CREATE DATABASE absensi_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'absensi'@'localhost' IDENTIFIED BY 'absensi';
GRANT ALL PRIVILEGES ON absensi_db.* TO 'absensi'@'localhost';
```

### 3. Konfigurasi environment

Salin dan sesuaikan variabel berikut (bisa via file `.env` + export, atau langsung di shell):

| Variabel | Wajib | Default | Keterangan |
|---|---|---|---|
| `PORT` | Tidak | `8080` | Port HTTP server |
| `DB_DSN` | Tidak | `absensi:absensi@tcp(127.0.0.1:3306)/absensi_db?parseTime=true` | DSN MySQL |
| `JWT_SECRET` | **Ya** | — | Kunci HMAC token (string acak panjang) |
| `JWT_ACCESS_TTL` | Tidak | `900` | Umur access token (detik) |
| `JWT_REFRESH_TTL` | Tidak | `604800` | Umur refresh token (detik) |
| `FACE_MATCH_THRESHOLD` | Tidak | `0.50` | Threshold cosine distance (match jika distance < threshold) |
| `FACE_LIVENESS_THRESHOLD` | Tidak | `0.70` | Skor liveness minimum yang diterima |
| `FACE_LIVENESS_REQUIRED` | Tidak | `true` | `false` untuk menonaktifkan penegakan liveness |
| `MAX_LIVENESS_FAILURES` | Tidak | `5` | Kegagalan liveness pemicu lockout |
| `LIVENESS_LOCKOUT_MINUTES` | Tidak | `15` | Lama kunci verifikasi wajah (menit) |

Contoh:

```bash
export JWT_SECRET="ganti-dengan-string-acak-minimal-32-karakter"
export DB_DSN="absensi:absensi@tcp(127.0.0.1:3306)/absensi_db?parseTime=true"
```

> Jangan pernah commit `JWT_SECRET` atau kredensial ke repo.

### 4. Jalankan migrasi

```bash
migrate -path migrations -database "mysql://absensi:absensi@tcp(127.0.0.1:3306)/absensi_db" up
```

Urutan migrasi: `000001_init` (users, employees, devices, attendance_logs) → `000002_face` (face_enrollments) → `000003_antispoof` (spoof_attempts). Untuk rollback satu langkah: ganti `up` dengan `down 1`.

### 5. Seed data awal (opsional, untuk development)

```bash
mysql absensi_db < migrations/seeds/seed.sql
```

Membuat akun operator **admin@example.com / admin123** dan 3 karyawan contoh. Ganti/hapus kredensial ini sebelum dipakai di lingkungan bersama.

### 6. Jalankan server

```bash
go run ./cmd/api
# atau: go build -o bin/api ./cmd/api && ./bin/api
```

Cek kesehatan: `curl localhost:8080/health` → `{"data":{"status":"ok"}}`.

### 7. Jalankan test

```bash
# Unit + handler test (tanpa database)
go test ./...

# Dengan vet & format check
go vet ./... && gofmt -l .
```

**Integration test** (lapisan repository, butuh MySQL asli — gunakan database khusus, jangan database development):

```bash
mysql -e "CREATE DATABASE absensi_test;"
INTEGRATION_DB_DSN="root@tcp(127.0.0.1:3306)/absensi_test?parseTime=true&multiStatements=true" \
  go test -tags=integration ./internal/repository/
```

Test integrasi otomatis menaikkan/menurunkan migrasi pada skema kosong dan membersihkan tabel setelah selesai.

## Daftar Endpoint

Basis URL: `http://localhost:8080/api/v1` (kecuali `/health`). Semua respons memakai envelope `{"data": ...}` atau `{"error": {"code", "message"}}`.

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| GET | `/health` | — | Cek kesehatan server |
| POST | `/api/v1/auth/register` | — | Daftarkan operator (name, email, password, role) |
| POST | `/api/v1/auth/login` | — | Login → access + refresh token |
| POST | `/api/v1/auth/refresh` | — | Tukar refresh token menjadi token baru |
| POST | `/api/v1/employees` | JWT | Tambah karyawan |
| GET | `/api/v1/employees` | JWT | Daftar karyawan (`?active_only`, `?limit`, `?offset`) |
| POST | `/api/v1/attendance/check-in` | JWT | Check-in (tolak jika sudah check-in / 409) |
| POST | `/api/v1/attendance/check-out` | JWT | Check-out (404 jika tidak ada check-in terbuka) |
| GET | `/api/v1/attendance` | JWT | Riwayat kehadiran (`?employee_id`, `?from`, `?to`) |
| POST | `/api/v1/face/enroll` | JWT | Enroll template wajah (1–5 embedding + quality_scores) → 201 |
| POST | `/api/v1/face/verify` | JWT | Verifikasi wajah 1:1 + liveness; 423 jika terkunci (lihat bawah) |
| GET | `/api/v1/face/challenge` | JWT | Terbitkan tantangan liveness sekali pakai |

### Contoh alur verify wajah (Milestone 3)

```bash
# 1. Login
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' | jq -r .data.access_token)

# 2. Minta tantangan liveness
CHALLENGE=$(curl -s localhost:8080/api/v1/face/challenge \
  -H "Authorization: Bearer $TOKEN" | jq -r .data.challenge_id)

# 3. Aplikasi mobile: lakukan tantangan (blink/putar kepala),
#    hitung liveness_score on-device, kirim hasil verify
curl -s -X POST localhost:8080/api/v1/face/verify \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"employee_id":1,"embedding":[...],"liveness_score":0.95,
       "device_id":"dev-1","challenge_id":"'"$CHALLENGE"'"}'
```

Respons sukses:

```json
{"data":{"match":true,"distance":0.12,"threshold":0.5,
         "liveness":{"score":0.95,"threshold":0.7,"passed":true}}}
```

Liveness gagal → `{"data":{"match":false,"threshold":0.5,"liveness":{"score":0.3,"threshold":0.7,"passed":false},"reject_reason":"liveness_too_low"}}` (HTTP 200, tercatat di `spoof_attempts`).

Terkunci → HTTP 423: `{"data":{"locked":true,"retry_after":840,"message":"face verification locked after repeated failed liveness checks; retry in 840 seconds"}}`.

## Struktur Direktori

```
backend/
├── cmd/api/            # main.go: wiring dependency & routing (chi)
├── internal/
│   ├── config/         # Konfigurasi dari environment variable
│   ├── model/          # Struct domain (User, Employee, FaceEnrollment, ...)
│   ├── face/           # Fungsi murni: cosine distance, rata-rata & codec embedding
│   ├── repository/     # Akses MySQL (users, employees, attendance, face, spoof)
│   ├── service/        # Logika bisnis + anti-spoofing (liveness, lockout, challenge)
│   ├── handler/        # Handler HTTP tipis + envelope JSON konsisten
│   ├── middleware/     # Request ID, logging, recovery, auth JWT
│   └── auth/           # Penerbitan & validasi token JWT
├── migrations/         # Migrasi SQL berversi (up/down) + seeds/
├── api/
│   ├── openapi.yaml    # Spesifikasi OpenAPI 3 (selalu sinkron dengan handler)
│   └── attendance.http # Koleksi REST Client siap jalan
└── README.md
```

## Catatan Operasional

- **Threshold adalah titik awal, bukan angka keramat.** `FACE_MATCH_THRESHOLD` dan `FACE_LIVENESS_THRESHOLD` harus dikalibrasi dengan data berlabel dari perangkat yang dipakai, lalu dilaporkan sebagai FAR/FRR. Jangan pernah menurunkan threshold hanya untuk "memperbaiki" kegagalan verifikasi — selidiki kualitas capture atau enroll ulang.
- **Challenge disimpan di memori** (map + mutex + TTL 2 menit). Cukup untuk satu instance; deployment multi-instance sebaiknya memakai Redis.
- **Data biometrik**: yang disimpan hanya embedding (BLOB), bukan foto. Terapkan kebijakan persetujuan (consent) dan retensi data sebelum production.
- Format respons error konsisten: `{"error":{"code":"bad_request|unauthorized|not_found|conflict|locked|internal_error","message":"..."}}`.
