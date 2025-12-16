# 📋 Forming - Production Monitoring Dashboard

Clean, organized production monitoring dashboard dengan integrasi Google Sheets dan date filtering.

## 📁 Structure

```
forming/
├── main.go              ← Main application
├── Dockerfile           ← Docker configuration
├── go.mod, go.sum       ← Dependencies
├── .env.local           ← Local config (gitignored)
│
├── lib/                 ← Helper libraries
│   ├── sheets.go        → Google Sheets integration
│   ├── skip_log.go      → Skip logging
│   └── date_filter.go   → Date filtering & prefixes
│
├── views/               ← HTML templates
│   ├── index.html
│   ├── data_list.html
│   ├── summary.html
│   └── skip_log.html
│
└── public/              ← Static assets
```

## 🚀 Quick Start

### Deploy dengan Docker (Recommended)

**⚠️ Gunakan docker-compose dari ROOT PROJECT:**

```bash
# Go to root
cd ..

# Start all services
.\dev\deploy-helper.ps1 start
```

Forming akan berjalan di: **http://localhost:3000**

### Local Development

```bash
# Install dependencies
go mod download

# Run
go run .
```

## ⚙️ Configuration

Edit `.env.local` atau environment variables:

```env
# Database
DB_HOST=postgres_db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=servfi

# MQTT
MQTT_HOST=emqx
MQTT_PORT=1883
MQTT_USER=servfi_app
MQTT_PASSWORD=S3cr3tP@ssw0rd!

# Google Sheets (Optional)
GOOGLE_SHEETS_CREDENTIALS=base64_encoded_json
GOOGLE_SPREADSHEET_ID=your_spreadsheet_id
GOOGLE_SHEET_NAME=Production Data
```

## ✨ Features

- ✅ Real-time MQTT subscription
- ✅ PostgreSQL data storage
- ✅ Google Sheets auto-export
- ✅ Date range filtering
- ✅ Status filtering (OK/Under/Over)
- ✅ CSV export
- ✅ Skip logging for invalid data
- ✅ Multi-line/prefix support

## 📚 Documentation

### Main Project Docs

- **[../README.md](../README.md)** - System overview
- **[../docs/INTEGRATION-GUIDE.md](../docs/INTEGRATION-GUIDE.md)** - Integration guide
- **[../docs/GOOGLE-SHEETS-SETUP.md](../docs/GOOGLE-SHEETS-SETUP.md)** - Google Sheets setup
- **[../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)** - Deployment (forming specific)

### Dev Tools

- **[../dev/tunnel.bat](../dev/tunnel.bat)** - SSH tunnel for development
- **[../dev/start-with-tunnel.bat](../dev/start-with-tunnel.bat)** - Start with tunnel

## 🛠️ Tech Stack

- **Backend**: Go + Fiber
- **Frontend**: HTMX + Alpine.js
- **Database**: PostgreSQL
- **MQTT**: Eclipse Paho
- **Cloud**: Google Sheets API

## 📊 Data Flow

```
IoT Device (ESP32)
    ↓ MQTT: production/mdcw
EMQX Broker
    ↓
Forming App
    ├→ PostgreSQL (storage)
    ├→ Google Sheets (export)
    └→ Web Dashboard (display)
```

## 🔧 Development

```bash
# Build
go build

# Run
go run .

# Test
go test ./...
```

## 📝 API Endpoints

```
GET /                          # Dashboard
GET /data-list                 # Data table
GET /summary                   # Production summary
GET /data-by-prefix?prefix=A1  # Filter by prefix
GET /data-by-date?start=...    # Date range filter
GET /skip-log                  # Skipped data log
GET /prefixes                  # List all prefixes
```

---

**Port**: 3000  
**MQTT Topic**: `production/mdcw`  
**Database Table**: `production_mdcw`
