# Absensi Face Recognition — Backend (Milestone 1)

REST API backend (Golang + MySQL) untuk aplikasi absensi pegawai.
Milestone 1: API absensi inti — auth JWT, data pegawai, check-in/check-out manual.
Face enrollment/verification + anti-spoofing menyusul di Milestone 2.

## Struktur

```
backend/
├── cmd/api/main.go                 # entrypoint: wiring + router
├── internal/
│   ├── config/                     # konfigurasi dari environment variables
│   ├── model/                      # domain entities
│   ├── auth/                       # JWT issue/parse (dipakai service + middleware)
│   ├── repository/                 # akses MySQL (database/sql)
│   ├── service/                    # business logic (depend on interfaces)
│   ├── handler/                    # HTTP handlers, thin
│   └── middleware/                 # request id, logging, recovery, JWT auth
├── migrations/                     # migrasi SQL ala golang-migrate (up/down)
│   └── seeds/seed.sql              # data contoh untuk development lokal
└── api/
    ├── openapi.yaml                # spesifikasi OpenAPI 3 (selalu sinkron dengan handler)
    └── attendance.http             # koleksi REST Client (VS Code) — happy path
```

## Prasyarat

- Go 1.22+
- MySQL 8.x
- (opsional) `migrate` CLI dari golang-migrate untuk menjalankan migrasi

## Setup lokal

```bash
# 1. Database + user
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS absensi_db;"
mysql -u root -p -e "CREATE USER IF NOT EXISTS 'absensi'@'localhost' IDENTIFIED BY 'absensi'; GRANT ALL ON absensi_db.* TO 'absensi'@'localhost';"

# 2. Migrasi (butuh migrate CLI: https://github.com/golang-migrate/migrate)
migrate -path migrations -database "mysql://absensi:absensi@tcp(127.0.0.1:3306)/absensi_db" up
# rollback bila perlu:
# migrate -path migrations -database "mysql://absensi:absensi@tcp(127.0.0.1:3306)/absensi_db" down

# 3. Seed (opsional, untuk development)
mysql -u absensi -pabsensi absensi_db < migrations/seeds/seed.sql
# demo login: admin@example.com / admin123

# 4. Environment & run
export JWT_SECRET="ganti-dengan-secret-yang-kuat"
export DB_DSN="absensi:absensi@tcp(127.0.0.1:3306)/absensi_db?parseTime=true"
export PORT="8080"
go run ./cmd/api
```

## Testing

```bash
# Unit + handler test (tanpa database)
go build ./... && go vet ./... && go test ./...

# Integration test repository (butuh MySQL + database KHUSUS test)
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS absensi_test;"
INTEGRATION_DB_DSN="root@tcp(127.0.0.1:3306)/absensi_test?parseTime=true&multiStatements=true" \
  go test -tags=integration ./internal/repository/
```

## Ringkasan API

Base URL: `http://localhost:8080` — lihat `api/openapi.yaml` untuk spesifikasi penuh
dan `api/attendance.http` untuk koleksi yang bisa langsung di-run dari VS Code.

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| GET | `/health` | — | Health check |
| POST | `/api/v1/auth/register` | — | Registrasi operator → token pair |
| POST | `/api/v1/auth/login` | — | Login → token pair |
| POST | `/api/v1/auth/refresh` | — | Tukar refresh token → token pair baru |
| GET | `/api/v1/employees` | JWT | List pegawai (`active_only`, `limit`, `offset`) |
| POST | `/api/v1/employees` | JWT | Tambah pegawai |
| POST | `/api/v1/attendance/check-in` | JWT | Check-in (`employee_no`, `method`, `device_id?`) |
| POST | `/api/v1/attendance/check-out` | JWT | Check-out (butuh check-in yang masih terbuka) |
| GET | `/api/v1/attendance` | JWT | Riwayat (`employee_id`, `from?`, `to?`) |

Semua response memakai envelope JSON:

```json
{ "data": { ... } }
{ "error": { "code": "bad_request", "message": "..." } }
```

## Aturan bisnis penting

- Check-in ganda tanpa check-out ditolak (`409 already checked in`).
- Check-out tanpa check-in terbuka ditolak (`404 no open check-in`).
- Check-out mewarisi `method` dari check-in pasangannya.
- `method: "face"` sudah disiapkan di skema & validasi, tapi verifikasi wajah
  baru aktif di Milestone 2; untuk sekarang check-in manual = terverifikasi operator.
- Semua write yang melibatkan >1 statement (check-in/out + pencatatan device)
  berjalan dalam satu transaksi database.
