# Template Environment Variables (.env)

Dokumen ini berisi template lengkap untuk file `.env` yang digunakan oleh semua service dalam proyek ini.

## 📋 Cara Penggunaan

1. Copy template di bawah ini ke file `.env` di root directory (`d:\iot\apps\.env`)
2. Sesuaikan nilai-nilai variable sesuai dengan konfigurasi Anda
3. **PENTING**: Jangan commit file `.env` ke repository (sudah ada di `.gitignore`)

---

## 🔧 Template .env

```env
# ============================================
# DATABASE CONFIGURATION
# ============================================
# PostgreSQL credentials used by all services
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
DB_NAME=iot_database
DB_HOST=postgres_db
DB_PORT=5432

# ============================================
# MQTT BROKER CONFIGURATION
# ============================================
# MQTT credentials for service authentication
# Used by: backend (forwarder), forming

# Primary MQTT credentials (used in docker-compose.yml hardcoded)
MQTT_USER=servfi_app
MQTT_PASSWORD=S3cr3tP@ssw0rd!

# Backend/Forwarder MQTT Configuration
MQTT_BROKER_URI=tcp://emqx:1883
MQTT_TOPIC=production/mdcw
MQTT_USERNAME=servfi_app
MQTT_PASSWORD=S3cr3tP@ssw0rd!

# Forwarder MQTT Publisher Configuration (if forwarding to external broker)
MQTT_BROKER_PUB=tcp://external-broker:1883
MQTT_TOPIC_PUB=production/forwarded
MQTT_USERNAME_FOR=forwarder_user
MQTT_PASSWORD_FOR=forwarder_password

# ============================================
# GOOGLE SHEETS INTEGRATION (OPTIONAL)
# ============================================
# Used by: forming service
# Leave empty if not using Google Sheets integration

# Base64-encoded Google Service Account JSON credentials
GOOGLE_SHEETS_CREDENTIALS=

# Google Spreadsheet ID (from URL)
# Example: https://docs.google.com/spreadsheets/d/SPREADSHEET_ID_HERE/edit
GOOGLE_SPREADSHEET_ID=

# Sheet name within the spreadsheet
GOOGLE_SHEET_NAME=Production Data

# ============================================
# TELEGRAM NOTIFICATION (OTA Service)
# ============================================
# Used by: ota service
# These are already hardcoded in docker-compose.yml
# Included here for reference/override if needed
TELEGRAM_BOT_TOKEN=8562677403:AAEp3bAEBbpqTIIMoxCA5XlIGUwYD3ZUtoc
TELEGRAM_CHAT_ID=7412135090
TELEGRAM_CONTACT_WA=+628123456789

# ============================================
# APPLICATION SETTINGS
# ============================================
# General application configuration

# Timezone for all services
TZ=Asia/Jakarta

# Server port (optional, can override defaults)
PORT=8888
```

---

## 📊 Daftar Service dan Environment Variables

### 1. **PostgreSQL** (`postgres`)

Environment variables yang digunakan:

- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `TZ` - Timezone (hardcoded: Asia/Jakarta)
- `PGTZ` - PostgreSQL timezone (hardcoded: Asia/Jakarta)

**Sumber**: Diambil dari `.env` melalui docker-compose.yml

---

### 2. **EMQX** (`emqx`)

Environment variables yang digunakan:

- `EMQX_ALLOW_ANONYMOUS=false` (hardcoded)
- `EMQX_AUTH__USER__1__USERNAME=servfi_app` (hardcoded)
- `EMQX_AUTH__USER__1__PASSWORD=S3cr3tP@ssw0rd!` (hardcoded)

**Sumber**: Hardcoded di docker-compose.yml

**Catatan**: EMQX tidak menggunakan `.env` file. Untuk mengubah credentials EMQX, edit docker-compose.yml dan pastikan kredensial yang sama digunakan di service lain.

---

### 3. **Backend/Forwarder** (`backend`)

Environment variables yang digunakan:

- `DB_HOST` - Database host (hardcoded: postgres_db)
- `DB_PORT` - Database port
- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `MQTT_BROKER_URI` - MQTT broker URI (hardcoded: tcp://emqx:1883)
- `MQTT_TOPIC` - MQTT topic to subscribe
- `MQTT_USERNAME` - MQTT username
- `MQTT_PASSWORD` - MQTT password
- `MQTT_BROKER_PUB` - External MQTT broker for forwarding (optional)
- `MQTT_TOPIC_PUB` - External MQTT topic for publishing (optional)
- `MQTT_USERNAME_FOR` - External MQTT username (optional)
- `MQTT_PASSWORD_FOR` - External MQTT password (optional)
- `TELEGRAM_CONTACT_WA` - WhatsApp contact for notifications
- `PORT` - Server port (default: 8888)
- `TZ` - Timezone (hardcoded: Asia/Jakarta)

**Sumber**: Menggunakan `env_file: .env` + environment variables di docker-compose.yml

**File konfigurasi**:

- `internal/database/database.go` - Database connection
- `internal/mqtt/client.go` - MQTT client configuration
- `internal/forwarder/forwarder.go` - Message forwarding
- `internal/config/config.go` - General configuration
- `main.go` - Port configuration

---

### 4. **OTA** (`ota`)

Environment variables yang digunakan:

- `MQTT_BROKER` - MQTT broker URI (hardcoded: tcp://emqx:1883)
- `MQTT_USER` - MQTT username (hardcoded: cntrl)
- `MQTT_PASS` - MQTT password (hardcoded: empty)
- `TELEGRAM_BOT_TOKEN` - Telegram bot token
- `TELEGRAM_CHAT_ID` - Telegram chat ID for notifications

**Sumber**: Hardcoded di docker-compose.yml

**Catatan**: OTA service tidak menggunakan `.env` file. Semua konfigurasi hardcoded di docker-compose.yml.

---

### 5. **Forming** (`forming`)

Environment variables yang digunakan:

- `DB_HOST` - Database host (hardcoded: postgres_db)
- `DB_PORT` - Database port (hardcoded: 5432)
- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `MQTT_HOST` - MQTT broker host (hardcoded: emqx)
- `MQTT_PORT` - MQTT broker port (hardcoded: 1883)
- `MQTT_USER` - MQTT username (hardcoded: servfi_app)
- `MQTT_PASSWORD` - MQTT password (hardcoded: S3cr3tP@ssw0rd!)
- `GOOGLE_SHEETS_CREDENTIALS` - Base64-encoded Google Service Account JSON
- `GOOGLE_SPREADSHEET_ID` - Google Spreadsheet ID
- `GOOGLE_SHEET_NAME` - Sheet name (hardcoded: Production Data)
- `TZ` - Timezone (hardcoded: Asia/Jakarta)

**Sumber**: Menggunakan environment variables di docker-compose.yml dengan nilai dari `.env`

**File konfigurasi**:

- `main.go` (lines 57-61) - Database configuration
- `main.go` (lines 82-85) - MQTT configuration
- `lib/sheets.go` (lines 25, 44-45) - Google Sheets configuration

---

## ⚙️ Variabel yang Harus Diubah

### 🔴 **CRITICAL** (Harus diubah sebelum production)

1. **`DB_PASSWORD`**

   - Default: `your_secure_password_here`
   - Gunakan password yang kuat (minimal 16 karakter, kombinasi huruf, angka, simbol)
   - Contoh: `P@ssw0rd!2024$SecureDB`

2. **`MQTT_PASSWORD`**
   - Default: `S3cr3tP@ssw0rd!`
   - **PENTING**: Harus sama dengan password di EMQX (hardcoded di docker-compose.yml)
   - Jika mengubah ini, ubah juga di:
     - `docker-compose.yml` → `emqx` → `EMQX_AUTH__USER__1__PASSWORD`
     - `docker-compose.yml` → `forming` → `MQTT_PASSWORD`

### 🟡 **RECOMMENDED** (Sebaiknya diubah)

1. **`DB_NAME`**

   - Default: `iot_database`
   - Sesuaikan dengan nama database yang Anda inginkan
   - Contoh: `servfi`, `production_db`, `iot_monitoring`

2. **`DB_USER`**
   - Default: `postgres`
   - Recommended: Buat user khusus untuk aplikasi
   - Contoh: `servfi_user`, `iot_app`

### 🟢 **OPTIONAL** (Tergantung fitur yang digunakan)

1. **Google Sheets Integration**

   - `GOOGLE_SHEETS_CREDENTIALS` - Wajib jika menggunakan Google Sheets
   - `GOOGLE_SPREADSHEET_ID` - Wajib jika menggunakan Google Sheets
   - Jika tidak menggunakan, biarkan kosong

2. **External MQTT Forwarding**

   - `MQTT_BROKER_PUB` - Jika ingin forward data ke broker eksternal
   - `MQTT_TOPIC_PUB` - Topic untuk publishing ke broker eksternal
   - `MQTT_USERNAME_FOR` - Username untuk broker eksternal
   - `MQTT_PASSWORD_FOR` - Password untuk broker eksternal

3. **Telegram Notifications**
   - `TELEGRAM_BOT_TOKEN` - Sudah dikonfigurasi di docker-compose.yml
   - `TELEGRAM_CHAT_ID` - Sudah dikonfigurasi di docker-compose.yml
   - `TELEGRAM_CONTACT_WA` - Nomor WhatsApp untuk kontak

---

## 🔄 Hirarki Prioritas Environment Variables

Jika sebuah variable didefinisikan di beberapa tempat, urutan prioritasnya:

1. **Environment variables di docker-compose.yml** (prioritas tertinggi)
2. **File .env** (melalui `env_file`)
3. **Default values di kode aplikasi** (prioritas terendah)

### Contoh:

```yaml
# docker-compose.yml
environment:
  - MQTT_HOST=emqx # ← Ini akan digunakan
env_file:
  - .env # MQTT_HOST di .env diabaikan jika sudah ada di environment
```

---

## 🧪 Validasi Konfigurasi

Setelah membuat file `.env`, validasi konfigurasi dengan:

```bash
# Validasi docker-compose syntax
docker-compose config --quiet

# Lihat environment variables yang akan digunakan
docker-compose config

# Test database connection
docker-compose up postgres -d
docker-compose exec postgres psql -U $DB_USER -d $DB_NAME -c "SELECT 1;"
```

---

## 🔐 Keamanan

**⚠️ PENTING**:

1. ✅ File `.env` sudah ada di `.gitignore` - JANGAN hapus dari gitignore
2. ✅ JANGAN commit file `.env` ke repository
3. ✅ Gunakan password yang kuat untuk production
4. ✅ Simpan backup `.env` di tempat yang aman (password manager, encrypted storage)
5. ✅ Rotasi password secara berkala (minimal setiap 3-6 bulan)

---

## 📝 Troubleshooting

### Database connection error

- Periksa `DB_USER`, `DB_PASSWORD`, `DB_NAME` sudah benar
- Pastikan service postgres sudah running: `docker-compose ps postgres`
- Check logs: `docker-compose logs postgres`

### MQTT connection error

- Periksa `MQTT_PASSWORD` sama dengan konfigurasi di EMQX
- Pastikan EMQX sudah running: `docker-compose ps emqx`
- Check logs: `docker-compose logs emqx`
- Akses EMQX dashboard: http://localhost:18083 (admin/public)

### Google Sheets error

- Pastikan `GOOGLE_SHEETS_CREDENTIALS` valid (base64-encoded JSON)
- Pastikan `GOOGLE_SPREADSHEET_ID` benar
- Check service account permissions di Google Cloud Console

---

## 📞 Kontak

Jika ada pertanyaan atau masalah terkait konfigurasi, silakan hubungi tim DevOps.

---

**Terakhir diperbarui**: 2025-12-15
