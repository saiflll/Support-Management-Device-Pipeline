# 🔐 Database Access Security - pgweb Implementation

## ✅ Yang Sudah Diimplementasikan

### 1. **pgweb Database Interface**
Mengganti Adminer dengan pgweb untuk interface database yang lebih modern dan optimized untuk PostgreSQL.

**Keunggulan pgweb:**
- ✅ Lebih ringan dan cepat
- ✅ Fokus untuk PostgreSQL
- ✅ Query history
- ✅ Export ke CSV, JSON, XML
- ✅ UI lebih modern dan mudah dikustomisasi

### 2. **Custom UI dengan Tema OTA**
Interface database dibuat matching dengan tema OTA dashboard:
- 🎨 Cyber-styled design
- 🌈 Animated backgrounds (morphing blobs)
- ✨ Smooth transitions dan hover effects
- 🔵 Cyan accent color (sama dengan OTA)
- 📱 Responsive design

### 3. **Security Implementation**
- 🔐 Session-based authentication
- 🔒 HTTPOnly & SameSite cookies
- 🚫 No direct access ke pgweb
- ⏱️ 24-hour session expiry
- 🔄 Auto-redirect ke login jika invalid

## 🎨 Tampilan UI

### Header Section
```
┌─────────────────────────────────────────────────────┐
│ 🗄️ Database Management                    [Close]  │
│ PostgreSQL Web Interface • Secure Access            │
└─────────────────────────────────────────────────────┘
```

### Info Cards
```
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│  Database   │ │   Server    │ │   Status    │
│   servfi    │ │PostgreSQL 13│ │● Connected  │
└─────────────┘ └─────────────┘ └─────────────┘
```

### Main Interface
```
┌─────────────────────────────────────────────────────┐
│ Database Explorer                      [🔄 Refresh] │
├─────────────────────────────────────────────────────┤
│                                                     │
│              [pgweb Interface Embedded]             │
│                                                     │
│  - Tables List                                      │
│  - Query Editor                                     │
│  - Results View                                     │
│  - Export Options                                   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## 🔒 Cara Kerja Keamanan

### Flow Diagram:
```
User → Login OTA/Forwarder
         ↓
   Telegram 2FA
         ↓
   Session Created (24h)
         ↓
   Click "Database" Button
         ↓
   pgweb Proxy Check Session
         ↓
    ┌────┴────┐
    │         │
  Valid?   Invalid
    │         │
    ▼         ▼
Custom UI   Redirect
+ pgweb     to Login
```

## 📋 Langkah Penggunaan

### 1. Login
```
OTA: http://localhost:9999
Forwarder: http://localhost:8888
```

### 2. Request Kode Verifikasi
- Klik "Minta Kode Baru"
- Kode 6 digit dikirim ke Telegram
- Berlaku 5 menit

### 3. Input Kode
- Masukkan kode dari Telegram
- Session dibuat (berlaku 24 jam)

### 4. Akses Database
- Klik tombol "Database" di header
- Custom UI terbuka di tab baru
- pgweb interface embedded di dalam

## 🔧 Konfigurasi

### Docker Compose

```yaml
services:
  # pgweb - Internal Only
  pgweb:
    image: sosedoff/pgweb
    environment:
      - DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=disable
      - PGWEB_SESSIONS=1
    # No external ports

  # pgweb Proxy - With Custom UI
  pgweb-proxy:
    image: rennn/pgweb-proxy:latest
    build:
      context: ./pgweb
    ports:
      - "8080:8080"
    environment:
      - PGWEB_URL=http://pgweb:8081
      - OTA_URL=http://localhost:9999/login
```

### Environment Variables

```env
# pgweb Proxy
PGWEB_URL=http://pgweb:8081
OTA_URL=http://localhost:9999/login
PORT=8080

# Telegram (untuk login)
TELE_BOT_OTA=<bot_token>
TELE_BOT_ALRT=<bot_token>
TELEGRAM_CHAT_ID=<chat_id>
```

## 📊 Port Mapping

| Service | Port | Access | Auth |
|---------|------|--------|------|
| OTA | 9999 | Public | ✅ Telegram |
| Forwarder | 8888 | Public | ✅ Telegram |
| **pgweb Proxy** | **8080** | **Public** | **✅ Session** |
| **pgweb** | **-** | **Internal** | **-** |
| PostgreSQL | 5432 | Internal | - |

## 🚀 Deployment

### Build & Start

```bash
# Build semua services
docker-compose build

# Start services
docker-compose up -d

# Check status
docker-compose ps
```

### Verify

```bash
# Test pgweb proxy
curl http://localhost:8080/health
# Should return: OK

# Test redirect (tanpa session)
curl -I http://localhost:8080
# Should redirect to login

# Check logs
docker-compose logs -f pgweb-proxy
docker-compose logs -f pgweb
```

## 🎯 Features Comparison

### Adminer vs pgweb

| Feature | Adminer | pgweb |
|---------|---------|-------|
| **Database Support** | Multi-DB | PostgreSQL only ✅ |
| **UI Customization** | Limited | Highly customizable ✅ |
| **Performance** | Good | Better for PG ✅ |
| **Size** | ~500KB | ~10MB |
| **Query History** | ❌ | ✅ |
| **Export Formats** | SQL | CSV, JSON, XML ✅ |
| **Theme Matching** | ❌ | ✅ Custom UI |

**Pilihan kami: pgweb** ✅

## 🔍 Troubleshooting

### Problem: pgweb tidak loading

**Solusi:**
```bash
# Check pgweb container
docker logs pgweb

# Check connection
docker exec pgweb-proxy curl http://pgweb:8081

# Restart
docker-compose restart pgweb pgweb-proxy
```

### Problem: UI tidak muncul

**Solusi:**
```bash
# Check views directory
docker exec pgweb-proxy ls -la /app/views

# Rebuild
docker-compose build pgweb-proxy
docker-compose up -d pgweb-proxy
```

### Problem: Session expired terus

**Solusi:**
```bash
# Clear browser cookies
# Check timezone
docker exec pgweb-proxy printenv TZ

# Restart all auth services
docker-compose restart pgweb-proxy ota forwarder
```

## 📁 File Structure

```
pgweb/
├── main.go              # Proxy server dengan custom UI
├── Dockerfile           # Container configuration
├── go.mod              # Go dependencies
├── README.md           # Documentation
└── views/
    └── index.html      # Custom UI dengan tema OTA
```

## 🎨 Customization

### Mengubah Warna Tema

Edit `pgweb/views/index.html`:

```css
/* Ganti cyan dengan warna lain */
.text-glow-cyan {
    color: #YOUR_COLOR;
    text-shadow: 0 0 15px rgba(YOUR_RGB, 0.6);
}

.card-cyan {
    border: 1px solid rgba(YOUR_RGB, 0.4);
    box-shadow: 0 4px 30px rgba(YOUR_RGB, 0.1);
}
```

### Menambah Fitur Custom

Edit `pgweb/main.go`:

```go
// Tambah endpoint custom
app.Get("/api/stats", requireAuthOrRedirect, func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "tables": 10,
        "size": "5.2 MB",
    })
})
```

## ✅ Checklist Deployment

- [ ] Update `.env` dengan credentials yang benar
- [ ] Build: `docker-compose build`
- [ ] Start: `docker-compose up -d`
- [ ] Test login OTA
- [ ] Test login Forwarder
- [ ] Test akses database dari OTA
- [ ] Test akses database dari Forwarder
- [ ] Test direct access (harus redirect)
- [ ] Verify custom UI tampil dengan benar
- [ ] Test query di pgweb
- [ ] Test export data

## 🎉 Kesimpulan

**Sekarang database HANYA bisa diakses melalui:**
1. ✅ Tombol "Database" di OTA dashboard (setelah login)
2. ✅ Tombol "Database" di Forwarder dashboard (setelah login)
3. ❌ TIDAK bisa akses langsung tanpa session!

**Keunggulan implementasi:**
- ✅ UI matching dengan tema OTA
- ✅ Lebih cepat dan ringan (pgweb)
- ✅ Security dengan 2FA Telegram
- ✅ Session management yang proper
- ✅ Network isolation
- ✅ Query history dan export features

---

**Dibuat:** 2025-12-17  
**Versi:** 2.0 (pgweb)  
**Status:** ✅ Production Ready
