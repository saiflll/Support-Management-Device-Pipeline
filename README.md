# 🏭 IoT Production Monitoring System

Sistem monitoring produksi terintegrasi untuk IoT dengan 3 komponen utama:

## 🎯 Komponen Sistem

| Komponen       | Port        | Fungsi                                    | Status |
| -------------- | ----------- | ----------------------------------------- | ------ |
| **Forming**    | 3000        | Dashboard monitoring produksi real-time   | ✅     |
| **Forwarder**  | 8888        | Backend aggregasi & forward data ke cloud | ✅     |
| **OTA**        | 9999        | Over-The-Air firmware update untuk ESP32  | ✅     |
| **PostgreSQL** | 5432        | Database storage                          | ✅     |
| **EMQX**       | 1883, 18083 | MQTT Broker & Dashboard                   | ✅     |

## 🚀 Quick Start

### 1. Persiapan

**Setup Environment Configuration:**

```bash
# Copy template environment
cp docs/ENV-TEMPLATE.md .env

# Edit .env - SATU FILE untuk SEMUA services!
notepad .env
```

**⚠️ PENTING**: Edit minimal:

- `DB_PASSWORD` - Ganti dari default
- `MQTT_PASSWORD` - Sesuaikan dengan EMQX

File `.env` ini digunakan oleh SEMUA services (forming, forwarder, OTA).

### 2. Start System (Mudah!)

```powershell
# Gunakan helper script
.\dev\deploy-helper.ps1 start
```

<details>
<summary>Atau manual dengan docker-compose</summary>

```bash
# Build images
docker-compose build

# Start semua services
docker-compose up -d

# Check status
docker-compose ps
```

</details>

### 3. Akses Dashboard

- 📊 **Forming Dashboard**: http://localhost:3000
- 🔄 **Forwarder Status**: http://localhost:8888/forwarder
- 📡 **OTA Dashboard**: http://localhost:9999
- 🌐 **EMQX Dashboard**: http://localhost:18083 (admin/public)

## 📚 Dokumentasi

- 📖 **[docs/INTEGRATION-GUIDE.md](docs/INTEGRATION-GUIDE.md)** - Panduan lengkap integrasi & troubleshooting
- 🔧 **[docs/ENV-TEMPLATE.md](docs/ENV-TEMPLATE.md)** - Template konfigurasi environment
- 📡 **[docs/MQTT_TOPICS_AND_PAYLOADS.md](docs/MQTT_TOPICS_AND_PAYLOADS.md)** - Struktur MQTT topics
- 📄 **[docs/JSON_UNIVERSAL_SCHEMA_M1_M11.md](docs/JSON_UNIVERSAL_SCHEMA_M1_M11.md)** - Schema JSON payload
- 🔄 **[docs/OTA_DOCUMENTATION.md](docs/OTA_DOCUMENTATION.md)** - Dokumentasi OTA update

### Dokumentasi per Komponen

- **Forming**: [`forming/README.md`](forming/README.md), [`forming/DEPLOYMENT.md`](forming/DEPLOYMENT.md)
- **Forwarder**: [`forward/README.md`](forward/README.md)

## 🛠️ Helper Commands

```powershell
# Start system
.\dev\deploy-helper.ps1 start

# Check status
.\dev\deploy-helper.ps1 status

# View logs
.\dev\deploy-helper.ps1 logs

# Backup database
.\dev\deploy-helper.ps1 backup

# Stop system
.\dev\deploy-helper.ps1 stop

# Clean containers
.\dev\deploy-helper.ps1 clean
```

## 🔍 Monitoring

```bash
# Logs real-time
docker-compose logs -f

# Logs service tertentu
docker-compose logs -f forming
docker-compose logs -f backend
docker-compose logs -f ota

# Status containers
docker-compose ps

# Resource usage
docker stats
```

## 🔐 Security Checklist

- [ ] Ganti `DB_PASSWORD` di `.env`
- [ ] Sesuaikan `MQTT_PASSWORD` di `.env`
- [ ] Tambahkan user MQTT di EMQX Dashboard
- [ ] Untuk production: Enable SSL/TLS
- [ ] Untuk production: Gunakan reverse proxy dengan HTTPS

## 🗂️ Struktur Project

```
d:\iot\apps\
├── forming/              # Dashboard produksi
├── forward/              # Backend forwarder
├── ota/                  # OTA update server
├── dev/                  # Development scripts
│   ├── deploy-helper.ps1 # Deployment helper
│   └── mqtt_publisher.py # MQTT testing script
├── docs/                 # Dokumentasi lengkap
│   ├── SETUP-CHECKLIST.md
│   ├── INTEGRATION-GUIDE.md
│   ├── QUICK-REFERENCE.md
│   ├── ENV-TEMPLATE.md
│   └── ...
├── docker-compose.yml    # Orchestration utama
├── .env                  # Konfigurasi (gitignored)
└── README.md             # This file
```

## 🔄 Workflow Data

```
ESP32 Device
    ↓ (MQTT Publish)
    ↓ Topic: production/mdcw
    ↓
EMQX Broker ←→ [Auth: servfi_app]
    ↓
    ├→ Forming (Subscribe) → PostgreSQL → Google Sheets (optional)
    ├→ Forwarder (Process) → PostgreSQL → Forward to Cloud (optional)
    └→ OTA (Monitor devices)
```

## 📊 MQTT Topics

| Topic                | Direction | Publisher | Subscriber | Purpose         |
| -------------------- | --------- | --------- | ---------- | --------------- |
| `production/mdcw`    | →         | ESP32     | Forming    | Data produksi   |
| `nodes/{id}/status`  | →         | ESP32     | Forwarder  | Status device   |
| `nodes/{id}/monitor` | →         | ESP32     | Forwarder  | Monitoring data |
| `ota/{id}/update`    | ←         | OTA       | ESP32      | Firmware update |

## 🆘 Troubleshooting

### Service tidak bisa connect

```bash
# Check network
docker network ls
docker network inspect servfor_iot-net

# Restart service
docker-compose restart forming
```

### MQTT authentication error

1. Buka EMQX Dashboard: http://localhost:18083
2. Login: `admin` / `public`
3. Menu: **Access Control** → **Authentication**
4. Tambahkan user: `servfi_app` / `S3cr3tP@ssw0rd!`

### Database connection error

```bash
# Check PostgreSQL
docker-compose logs postgres

# Test connection
docker exec -it postgres_db psql -U postgres -d servfi
```

Lihat **[docs/INTEGRATION-GUIDE.md](docs/INTEGRATION-GUIDE.md)** untuk troubleshooting lengkap.

## 📝 Development

### Update service tertentu

```bash
# Rebuild dan restart
docker-compose build forming
docker-compose up -d forming
```

### Database migrations

```bash
# Masuk ke container
docker exec -it postgres_db psql -U postgres -d servfi

# Atau dari host
docker exec postgres_db psql -U postgres -d servfi -c "SELECT * FROM records LIMIT 5;"
```

## 🔧 Maintenance

### Backup

```bash
# Auto backup dengan helper
.\dev\deploy-helper.ps1 backup

# Manual backup
docker exec postgres_db pg_dump -U postgres servfi > backup_$(Get-Date -Format "yyyyMMdd").sql
```

### Update

```bash
# Pull latest changes
git pull

# Rebuild
docker-compose build

# Restart
docker-compose up -d
```

## 📞 Support

Untuk issue atau pertanyaan:

1. Check logs: `docker-compose logs -f [service]`
2. Baca **[docs/INTEGRATION-GUIDE.md](docs/INTEGRATION-GUIDE.md)**
3. Periksa konfigurasi di `.env`
4. Verifikasi MQTT credentials di EMQX Dashboard

---

**Version**: 1.0  
**Last Updated**: 2025-12-15
