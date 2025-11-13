# Deployment Fix Guide

## Problem yang Diperbaiki:

1. ✅ Error gob encoding untuk session (time.Time) - sudah fixed dengan menggunakan Unix timestamp
2. ✅ UI/UX improvements - morphism effect, 4 warna theme (purple, cyan, red, orange)
3. ✅ File action buttons - sekarang 3 tombol (rename, download, delete) dengan icon colorful
4. ✅ Node cards - warna orange untuk running nodes
5. ✅ Tab switching - sudah bekerja dengan baik
6. ✅ Login page - disesuaikan dengan theme

## Deployment di Server

### Problem: vendor directory tidak ada

Saat deploy di server, error terjadi karena Dockerfile mencoba COPY vendor tapi folder tidak ada.

### Solution:

#### Option 1: Generate vendor sebelum build (Recommended)

1. Di lokal Windows, jalankan:
```bash
cd forward
go mod vendor

cd ../ota
go mod vendor
```

2. Commit vendor folders ke git
```bash
git add forward/vendor ota/vendor
git commit -m "Add vendor directories"
git push
```

3. Deploy ke server seperti biasa

#### Option 2: Ubah Dockerfile agar tidak pakai vendor

**File: forward/Dockerfile**
```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o forwarder .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/forwarder .
EXPOSE 8090
CMD ["./forwarder"]
```

**File: ota/Dockerfile**
```dockerfile
# Build stage
FROM golang:1.25-alpine AS build

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN apk add --no-cache git
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o iot-ota-server .

# Runtime stage
FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=build /app/iot-ota-server .
COPY --from=build /app/views ./views
COPY --from=build /app/static ./static
EXPOSE 9999
CMD ["./iot-ota-server"]
```

## Install Go di Server (Jika Pakai Option 1)

Jika server belum punya Go:

```bash
# Download Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz

# Extract
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/bin/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

## Command untuk Deploy

```bash
# Di server
cd ~/public_html/ck3/tes/servfor

# Pull latest code
git pull

# Jika pakai Option 1 dan vendor sudah ada di git
docker-compose up -d --build

# Jika pakai Option 2 (Dockerfile sudah diubah)
docker-compose up -d --build

# Check logs
docker-compose logs -f
```

## Environment Variables (.env)

Pastikan file `.env` ada di root directory:

```env
# MQTT Config
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=ota-server

# Telegram Bot Config
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here

# Server Config
PORT=9999
```

## Monitoring

```bash
# Check status
docker-compose ps

# View logs
docker-compose logs -f ota-app
docker-compose logs -f forwarder

# Restart specific service
docker-compose restart ota-app
docker-compose restart forwarder
```

## Troubleshooting

### Issue: Cannot find Go command
```bash
# Install Go di server
sudo apt update
sudo apt install golang-go
```

### Issue: Permission denied
```bash
# Fix permissions
sudo chown -R $USER:$USER ~/public_html/ck3/tes/servfor
chmod +x deploy.sh
```

### Issue: Port already in use
```bash
# Check what's using the port
sudo lsof -i :9999
sudo lsof -i :8090

# Kill the process
sudo kill -9 <PID>
```

## Obfuscation & Security

Untuk production, HTML sudah include:
- Minified inline styles
- Single-file serving (mengurangi request)
- Session dengan expiry 5 menit untuk auth code
- HTTPS ready (tinggal setup reverse proxy)

Untuk lebih aman lagi, setup Nginx reverse proxy dengan SSL:

```nginx
server {
    listen 443 ssl http2;
    server_name ota.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:9999;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
