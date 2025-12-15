# 🚀 Integrasi Forming + Forwarder + OTA

Dokumentasi lengkap untuk menjalankan sistem IoT terintegrasi dengan 3 komponen utama:

1. **Forming App** - Dashboard monitoring produksi
2. **Forwarder** - Backend untuk aggregasi dan forward data ke cloud
3. **OTA** - Over-The-Air update server untuk ESP32

## 📊 Arsitektur Sistem

```
┌─────────────────────────────────────────────────────────────────┐
│                         IoT DEVICES (ESP32)                     │
│          Topic: production/mdcw, nodes/{id}/status, etc.        │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      EMQX MQTT Broker                           │
│             Port: 1883 (MQTT), 18083 (Dashboard)                │
└─────┬─────────────────┬─────────────────────┬───────────────────┘
      │                 │                     │
      ▼                 ▼                     ▼
┌──────────┐    ┌──────────────┐    ┌─────────────┐
│ FORMING  │    │  FORWARDER   │    │     OTA     │
│ Port:3000│    │  Port: 8888  │    │  Port: 9999 │
└────┬─────┘    └──────┬───────┘    └─────────────┘
     │                 │
     │                 │
     ▼                 ▼
┌─────────────────────────────────────┐
│      PostgreSQL Database             │
│         Port: 5432 (internal)        │
└─────────────────────────────────────┘
```

## 🔧 Komponen Sistem

### 1. **Forming App** (Port 3000)

- **Fungsi**: Dashboard untuk monitoring produksi real-time
- **Subscribe**: `production/mdcw`
- **Database**: Menyimpan data produksi ke PostgreSQL
- **Fitur**:
  - Real-time monitoring data produksi
  - Filter berdasarkan status (OK/UNDER/OVER)
  - Statistik per prefix produksi
  - Export ke Google Sheets (optional)

### 2. **Forwarder** (Port 8888)

- **Fungsi**: Backend untuk aggregasi data dan forward ke cloud eksternal
- **Subscribe**: Multiple topics dari IoT devices
- **Features**:
  - Aggregasi data setiap 5 menit
  - Forward ke MQTT publik/cloud (optional)
  - Dashboard status forwarder di `/forwarder`
  - Notifikasi Telegram

### 3. **OTA Server** (Port 9999)

- **Fungsi**: Over-The-Air firmware update untuk ESP32
- **Features**:
  - Upload firmware via web interface
  - Auto-detect ESP32 devices dari MQTT
  - Push firmware update via MQTT
  - Notifikasi Telegram

## 🚀 Quick Start

### 1. Persiapan Environment

Buat file `.env` di root folder (`d:\iot\apps\.env`):

```bash
# Copy dari template
cp ENV-TEMPLATE.md .env

# Edit sesuai kebutuhan
# Minimal yang wajib diisi:
# - DB_PASSWORD
# - MQTT_PASSWORD (jika berbeda)
```

### 2. Build dan Jalankan Semua Service

```bash
# Build semua images
docker-compose build

# Jalankan semua service
docker-compose up -d

# Cek status
docker-compose ps
```

### 3. Verifikasi Service Berjalan

```bash
# Cek logs
docker-compose logs -f forming
docker-compose logs -f backend
docker-compose logs -f ota

# Cek healthcheck
docker ps
```

### 4. Akses Dashboard

- **Forming Dashboard**: http://localhost:3000
- **Forwarder Dashboard**: http://localhost:8888/forwarder
- **OTA Dashboard**: http://localhost:9999
- **EMQX Dashboard**: http://localhost:18083 (user: admin, pass: public)

## 🔑 Konfigurasi MQTT Credentials

Pastikan pengguna MQTT sudah dibuat di EMQX Dashboard:

1. Akses: http://localhost:18083
2. Login dengan: `admin` / `public`
3. Menu: **Access Control** > **Authentication** > **Password-Based**
4. Tambahkan user:
   - Username: `servfi_app`
   - Password: `S3cr3tP@ssw0rd!`

## 📝 Environment Variables Detail

### Database Variables (Wajib)

```env
DB_HOST=postgres_db          # Hostname dalam Docker network
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password    # ⚠️ WAJIB DIGANTI!
DB_NAME=servfi
```

### MQTT Internal (Wajib)

```env
MQTT_BROKER_URI=tcp://emqx:1883
MQTT_USERNAME=servfi_app
MQTT_PASSWORD=S3cr3tP@ssw0rd!  # ⚠️ Harus sama dengan EMQX
```

### MQTT Public Forwarder (Optional)

```env
# Kosongkan jika tidak ingin forward ke external
MQTT_BROKER_PUB=tcp://public-broker.example.com:1883
MQTT_USERNAME_FOR=your_username
MQTT_PASSWORD_FOR=your_password
MQTT_TOPIC_PUB=sensor/data/ingest
```

### Google Sheets (Optional)

```env
# Untuk export data forming ke Google Sheets
GOOGLE_SHEETS_CREDENTIALS={"type":"service_account",...}
GOOGLE_SPREADSHEET_ID=your_spreadsheet_id
GOOGLE_SHEET_NAME=Production Data
```

Lihat `forming/GOOGLE-SHEETS-SETUP.md` untuk setup lengkap.

### Telegram Notifications (Optional)

```env
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

## 🔍 Troubleshooting

### Problem: Service tidak bisa connect ke database

```bash
# Cek apakah postgres sudah ready
docker-compose logs postgres

# Cek healthcheck
docker inspect postgres_db | grep Health

# Restart service yang bermasalah
docker-compose restart forming
```

### Problem: MQTT connection refused

```bash
# Cek EMQX logs
docker-compose logs emqx

# Verifikasi credentials di dashboard
# http://localhost:18083

# Restart EMQX
docker-compose restart emqx
```

### Problem: Forming tidak menerima data

```bash
# Cek apakah ESP32 publish ke topic yang benar
# Topic: production/mdcw

# Cek logs forming
docker-compose logs -f forming

# Test publish manual via EMQX dashboard:
# Topic: production/mdcw
# Payload:
{
  "ts": "2025-12-15 08:45:30",
  "reg2": 1,
  "reg5": 250,
  "reg114": 1,
  "prefix": "A1"
}
```

### Problem: Port sudah digunakan

```bash
# Cek port yang terpakai
netstat -ano | findstr "3000"    # Forming
netstat -ano | findstr "8888"    # Forwarder
netstat -ano | findstr "9999"    # OTA

# Ganti port di docker-compose.yml jika perlu
# Misal: "3001:3000" untuk expose port 3001 ke host
```

## 📦 Data Persistence

Data disimpan dalam Docker volumes:

- `postgres_data` - Database PostgreSQL
- `emqx_data` - Konfigurasi EMQX
- `emqx_log` - Log EMQX
- `./ota/static/uploads` - Firmware files (bind mount)

### Backup Database

```bash
# Backup
docker exec postgres_db pg_dump -U postgres servfi > backup_$(date +%Y%m%d).sql

# Restore
docker exec -i postgres_db psql -U postgres servfi < backup_20251215.sql
```

## 🔄 Update dan Maintenance

### Update Service Tertentu

```bash
# Rebuild service tertentu
docker-compose build forming
docker-compose up -d forming

# Rebuild semua
docker-compose build
docker-compose up -d
```

### Lihat Logs Real-time

```bash
# Semua service
docker-compose logs -f

# Service tertentu
docker-compose logs -f forming
docker-compose logs -f backend
docker-compose logs -f ota
```

### Stop dan Remove

```bash
# Stop semua
docker-compose stop

# Stop dan remove containers (data tetap aman di volumes)
docker-compose down

# Remove termasuk volumes (⚠️ DATA AKAN HILANG!)
docker-compose down -v
```

## 🌐 Network Configuration

Semua service terhubung dalam network `iot-net`:

- Service bisa saling komunikasi menggunakan container name
- Contoh: `postgres_db`, `emqx`, `forming-app`, `forwarder`, `ota-app`

## 📊 Monitoring

### Health Checks

- PostgreSQL: `pg_isready`
- EMQX: `emqx ping`
- Service otomatis wait sampai dependencies healthy

### Resource Usage

```bash
# Lihat resource usage
docker stats

# Lihat per container
docker stats forming-app forwarder ota-app
```

## 🔐 Security Notes

1. **⚠️ Ganti password default** di file `.env`
2. **Jangan commit** file `.env` ke git
3. **Gunakan strong password** untuk database dan MQTT
4. **Batasi akses port** hanya untuk development
5. Untuk production, gunakan:
   - SSL/TLS untuk MQTT
   - SSL untuk database connection
   - Reverse proxy (nginx) dengan HTTPS

## 📚 Dokumentasi Terkait

- **Forming**: `forming/README.md`, `forming/DEPLOYMENT.md`
- **Forwarder**: `forward/README.md`
- **OTA**: `OTA_DOCUMENTATION.md`
- **MQTT Topics**: `MQTT_TOPICS_AND_PAYLOADS.md`
- **JSON Schema**: `JSON_UNIVERSAL_SCHEMA_M1_M11.md`

## 🆘 Support

Jika ada masalah:

1. Cek logs: `docker-compose logs -f [service_name]`
2. Verifikasi konfigurasi di `.env`
3. Pastikan semua credentials sudah benar
4. Cek EMQX dashboard untuk MQTT issues
5. Periksa dokumentasi di folder masing-masing service

---

**Last Updated**: 2025-12-15
**Version**: 1.0
