# Menjalankan API di Laptop agar Bisa Diakses HP (Testing)

VM tempat saya berjalan tidak bisa membuka akses publik ke internet
(dibatasi proxy egress), jadi cara paling praktis untuk mengetes APK
adalah **menjalankan API di laptop/PC kamu sendiri** — HP dan laptop
cukup berada di **satu jaringan WiFi yang sama**.

## 1. Persiapan (sekali saja)

1. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
   (sudah termasuk `docker compose`).
2. Clone repo ini di laptop:
   `git clone https://github.com/ZepiDarmawanTambunan/app-absensi.git`
3. Masuk ke folder repo, salin contoh environment:
   `cp .env.example .env`
4. Isi `JWT_SECRET` di file `.env` dengan string acak panjang, misal hasil dari:
   `openssl rand -hex 32`

## 2. Jalankan

```bash
docker compose up --build -d
curl http://localhost:8080/health   # harus menjawab {"data":{"status":"ok"}}
```

Migrasi database dan data awal (admin + 3 karyawan contoh) diterapkan
otomatis saat pertama kali dijalankan.

## 3. Sambungkan HP

1. Cari **IP LAN laptop** (contoh `192.168.1.50`):
   - Windows: `ipconfig` → lihat *IPv4 Address* pada adapter WiFi
   - macOS/Linux: `ip addr` / `ifconfig`
2. Di HP, buka browser dan tes: `http://192.168.1.50:8080/health`
   (ganti dengan IP laptop kamu). Kalau OK, jaringan beres.
3. Di aplikasi Flutter, set base URL API ke IP tersebut:
   `API_BASE_URL=http://192.168.1.50:8080`
   (build dengan `--dart-define API_BASE_URL=...`).
4. Login awal: **admin@example.com** / **admin123** — segera ganti
   password-nya setelah masuk.

## 4. Perintah berguna

```bash
docker compose logs -f api      # lihat log API
docker compose down             # hentikan semua
docker compose down -v          # hentikan + HAPUS data database
```

## Catatan

- Setup ini untuk **testing internal**, bukan produksi.
- Jangan expose port 8080 laptop ke internet tanpa tahu risikonya.
- Kalau nanti butuh server yang benar-benar online 24 jam, opsi termudah
  adalah VPS murah lalu jalankan `docker compose` yang sama di sana.
