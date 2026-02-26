# 📋 Service Credentials & Access Guide
# Dokumen: dev/credentials.md
# ⚠️  CONFIDENTIAL — Jangan commit ke repo publik!

---

## 🌐 Service URLs

| Service | URL (Dev) | URL (Prod) |
|---------|-----------|------------|
| **OTA Dashboard** | http://localhost:9999 | https://your-domain:9999 |
| **Forming Monitor** | http://localhost:3000 | https://your-domain:3000 |
| **MQTT Broker (EMQX)** | mqtt://localhost:1883 | mqtt://your-domain:1883 |
| **PostgreSQL** | localhost:5432 | internal (via docker net) |

---

## 🔐 Forming Service — JWT Auth

> Service URL: Port **3000**
> Auth type: **JWT via POST /api/login**

| Field | Value |
|-------|-------|
| Username | `ppa3` |
| Password | `plan3ppa` |
| Token Expiry | 12 jam |
| Cookie Name | `forming_token` |

**Login Endpoint:**
```
POST /api/login
Content-Type: application/json

{ "username": "ppa3", "password": "plan3ppa" }
```

**Response:**
```json
{ "token": "<jwt>", "username": "ppa3" }
```

**Protected Endpoints (require Bearer token or cookie):**
- `GET /api/data?prefix=&status=&sort=&start_date=&end_date=`
- `GET /api/summary`
- `GET /api/prefixes`
- `GET /api/skip-log`
- `GET /api/export-csv?...`

---

## 🔑 OTA Dashboard — Auth

> Service URL: Port **9999**
> Auth: **Saat ini DISABLED** (semua request lolos)

Untuk re-enable:
1. Buka `ota/main.go`
2. Uncomment blok `requireAuth` di fungsi yang sama (line ~877)
3. Auth original pakai Telegram OTP via env var:
   - `TELE_BOT_OTA` — Telegram bot token
   - `TELEGRAM_CHAT_ID` — Chat ID admin
   - `WEBHOOK_TOKEN` — Token untuk webhook

---

## 🗄️ Database (PostgreSQL)

Diset via environment variables di `.env`:

```env
DB_HOST=postgres_db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<lihat .env>
DB_NAME=servfi
```

---

## 🐳 Docker Images (Docker Hub: rennnagge)

| Image | Tag | Service |
|-------|-----|---------|
| `rennnagge/ota-app` | latest | OTA + Dashboard Svelte |
| `rennnagge/monitor-app` | latest | Monitor Service |
| `rennnagge/forming-app` | latest | Forming Monitor |
| `rennnagge/postgres-db` | latest | DB Initializer |
| `rennn/forwarder-app` | latest | MQTT Forwarder |

---

## 📡 MQTT Default Credentials

```env
MQTT_USER=apps
MQTT_PASSWORD=apps
MQTT_PORT=1883
```

---

_Last updated: 2026-02-25_
