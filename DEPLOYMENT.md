# IoT OTA Server - Deployment Guide

## ✅ Masalah yang Sudah Diperbaiki

### 1. Session Storage Error (time.Time)
**Error**: `gob: type not registered for interface: time.Time`
**Solusi**: Session sekarang menyimpan Unix timestamp (int64) bukan time.Time
- File: `ota/main.go` line 501-502

### 2. File Actions Button
**Update**: Tombol file actions sekarang menggunakan layout 2x2 dengan 4 warna berbeda
- Purple: Copy Link
- Cyan: Rename
- Orange: Download
- Red: Delete
- File: `ota/static/app.js`

### 3. Dockerfile Dependency
**Masalah**: Server production tidak punya Go installed
**Solusi**: Dockerfile updated untuk auto-download dependencies
- File: `forward/Dockerfile`

## 🚀 Deploy ke Server

### 1. Pastikan Docker Installed di Server
```bash
docker --version
docker-compose --version
```

### 2. Upload Files ke Server
```bash
# Di local (Windows)
scp -r "d:\iot\suhu ck 3\server\serv-lokal" user@server:/path/to/destination

# Atau menggunakan Git
cd "d:\iot\suhu ck 3\server\serv-lokal"
git init
git add .
git commit -m "Initial commit"
git push origin main
```

### 3. Build dan Run di Server
```bash
# SSH ke server
ssh user@server

# Masuk ke directory project
cd /path/to/serv-lokal

# Build dan run dengan docker-compose
docker-compose up -d --build

# Check logs
docker-compose logs -f
```

### 4. Check Status
```bash
# Check running containers
docker-compose ps

# Check logs untuk debugging
docker-compose logs ota
docker-compose logs forwarder
```

## 🔒 Security Notes

### Environment Variables
Pastikan set environment variables di `.env` atau `docker-compose.yml`:
```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here
MQTT_BROKER=tcp://your_broker:1883
```

### Firewall Rules
```bash
# Allow port 9999 (OTA Server)
sudo ufw allow 9999/tcp

# Allow MQTT port if needed
sudo ufw allow 1883/tcp
```

## 🎨 UI Features

### Theme
- 4-Color Cyber Theme: Purple, Cyan, Red, Orange
- Animated morph blobs background
- Glass-morphism cards
- Smooth transitions

### Tabs
1. **OTA & Nodes**: Upload firmware, monitor nodes
2. **Forwarder**: MQTT forwarder status and buffer

### Authentication
- Telegram OTP verification
- Session-based authentication
- 6-digit code, valid for 5 minutes

## 📝 Usage

### 1. Login
- Klik "Kirim Kode ke Telegram"
- Check Telegram untuk kode verifikasi
- Masukkan kode 6 digit
- Login

### 2. Upload OTA Firmware
- Tab "OTA & Nodes"
- Choose file
- Click "Upload Firmware"

### 3. Monitor Nodes
- Tab "Running" untuk nodes yang online
- Tab "Offline" untuk nodes yang offline
- Auto-refresh every 5 seconds

### 4. Monitor Forwarder
- Tab "Forwarder"
- See buffer status
- See countdown to next forward
- View buffered data

## 🐛 Troubleshooting

### Container tidak start
```bash
# Check logs
docker-compose logs

# Rebuild
docker-compose down
docker-compose up --build -d
```

### Session error
- Pastikan `TELEGRAM_BOT_TOKEN` dan `TELEGRAM_CHAT_ID` sudah di-set
- Restart container jika perlu

### Port already in use
```bash
# Check what's using port 9999
netstat -tulpn | grep 9999

# Kill process or change port di docker-compose.yml
```

## 📦 Files Structure

```
serv-lokal/
├── ota/
│   ├── views/
│   │   ├── index.html       # Main dashboard
│   │   ├── login.html       # Login page
│   │   └── minify.html      # Minified version
│   ├── static/
│   │   └── app.js           # Frontend JavaScript
│   ├── main.go              # Backend Go code
│   ├── Dockerfile
│   └── go.mod
├── forward/
│   ├── internal/
│   ├── main.go
│   ├── Dockerfile
│   └── go.mod
└── docker-compose.yml
```

## 🔄 Update Process

```bash
# Pull latest changes
git pull

# Rebuild containers
docker-compose down
docker-compose up --build -d

# Check logs
docker-compose logs -f
```

## ⚡ Performance Tips

1. Use minified version for production (minify.html)
2. Enable CDN caching for Tailwind CSS
3. Use Redis instead of memory storage for sessions (high traffic)
4. Monitor container resources with `docker stats`

## 📞 Support

Jika ada masalah:
1. Check logs: `docker-compose logs`
2. Check container status: `docker-compose ps`
3. Restart services: `docker-compose restart`
4. Full rebuild: `docker-compose down && docker-compose up --build -d`
