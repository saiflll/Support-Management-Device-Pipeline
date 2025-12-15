# ============================================================

# INTEGRASI LENGKAP: FORMING + FORWARDER + OTA

# ============================================================

# File ini adalah template untuk konfigurasi semua service

# Copy file ini ke .env dan sesuaikan nilai-nilainya

# ============================================================

# DATABASE CONFIGURATION (PostgreSQL)

# ============================================================

# Digunakan oleh: backend (forwarder) dan forming

DB_HOST=postgres_db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password_rahasia_anda
DB_NAME=servfi

# ============================================================

# MQTT CONFIGURATION (EMQX)

# ============================================================

# Untuk Backend/Forwarder - Internal MQTT

MQTT_BROKER_URI=tcp://emqx:1883
MQTT_USERNAME=servfi_app
MQTT_PASSWORD=S3cr3tP@ssw0rd!

# Untuk Forwarder - MQTT Publik (Forward ke cloud/external)

# Kosongkan jika tidak ingin menggunakan fitur forward ke external

MQTT_BROKER_PUB=
MQTT_USERNAME_FOR=
MQTT_PASSWORD_FOR=
MQTT_TOPIC_PUB=sensor/data/ingest

# ============================================================

# TELEGRAM NOTIFICATION

# ============================================================

# Digunakan oleh: backend (forwarder) dan OTA

TELEGRAM_BOT_TOKEN=8562677403:AAEp3bAEBbpqTIIMoxCA5XlIGUwYD3ZUtoc
TELEGRAM_CHAT_ID=7412135090

# ============================================================

# GOOGLE SHEETS INTEGRATION (OPTIONAL)

# ============================================================

# Digunakan oleh: forming (untuk export data ke Google Sheets)

# Kosongkan jika tidak ingin menggunakan fitur Google Sheets

GOOGLE_SHEETS_CREDENTIALS=
GOOGLE_SPREADSHEET_ID=
GOOGLE_SHEET_NAME=Production Data

# ============================================================

# SERVICE PORTS

# ============================================================

# Backend/Forwarder: 8888

# Forming: 3000

# OTA: 9999

# PostgreSQL: 5432 (internal only)

# EMQX MQTT: 1883

# EMQX Dashboard: 18083
