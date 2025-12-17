# pgweb Proxy - Secure Database Access with Custom UI

## Overview

pgweb Proxy adalah service yang mengamankan akses ke pgweb (PostgreSQL web interface) dengan memerlukan autentikasi session yang valid dari OTA atau Forwarder dashboard. Interface dibuat dengan tema yang matching dengan OTA dashboard untuk konsistensi visual.

## Kenapa pgweb?

pgweb dipilih karena:
- ✅ **Lebih ringan** dibanding Adminer
- ✅ **Fokus PostgreSQL** - optimized untuk PostgreSQL
- ✅ **Modern UI** - lebih mudah dikustomisasi
- ✅ **Better performance** - lebih cepat untuk query besar
- ✅ **Query history** - menyimpan history query
- ✅ **Export data** - support export ke CSV, JSON, XML

## Custom UI Features

### Tema OTA Matching
- 🎨 **Cyber-styled design** dengan animated backgrounds
- 🌈 **Gradient effects** yang sama dengan OTA dashboard
- ✨ **Smooth animations** dan hover effects
- 🔵 **Cyan accent color** untuk konsistensi
- 📱 **Responsive design** untuk mobile dan desktop

### Security Features
- 🔐 **Session-based authentication**
- 🔒 **HTTPOnly & SameSite cookies**
- 🚫 **No direct access** - harus melalui OTA/Forwarder
- ⏱️ **24-hour session expiry**
- 🔄 **Auto-redirect** jika session invalid

## Architecture

```
User → OTA/Forwarder Login → Session Created
                                    ↓
User → Click "Database" → pgweb Proxy → Validate Session
                                              ↓
                                        Valid? → Custom UI + pgweb iframe
                                              ↓
                                        Invalid? → Redirect to Login
```

## Configuration

### Environment Variables

```env
PGWEB_URL=http://pgweb:8081          # Internal pgweb URL
OTA_URL=http://localhost:9999/login  # OTA login URL for redirect
PORT=8080                             # Proxy port
```

### pgweb Container

```yaml
pgweb:
  image: sosedoff/pgweb
  environment:
    - DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable
    - PGWEB_SESSIONS=1
```

## Usage

### From OTA Dashboard
1. Login to OTA (http://localhost:9999)
2. Click "Database" button
3. Custom UI opens with pgweb embedded

### From Forwarder Dashboard
1. Login to Forwarder (http://localhost:8888)
2. Click "Database" button
3. Custom UI opens with pgweb embedded

## Custom UI Components

### Header
- Database icon with cyber styling
- Connection status indicator
- Close button
- Matching OTA theme

### Info Cards
- Database name (servfi)
- Server version (PostgreSQL 13)
- Connection status (Connected/Disconnected)

### Main Interface
- Embedded pgweb iframe
- Refresh button
- Cyber-styled container
- Security indicator

## Development

### Build

```bash
cd pgweb
docker build -t rennn/pgweb-proxy:latest .
```

### Run Locally

```bash
cd pgweb
go run main.go
```

### Test

```bash
# Test health endpoint
curl http://localhost:8080/health

# Test redirect (without session)
curl -I http://localhost:8080
# Should redirect to login
```

## Customization

### Changing Theme Colors

Edit `views/index.html`:

```css
/* Change primary color from cyan to your color */
.text-glow-cyan {
    color: #YOUR_COLOR;
    text-shadow: 0 0 15px rgba(YOUR_RGB, 0.6);
}

.card-cyan {
    border: 1px solid rgba(YOUR_RGB, 0.4);
}
```

### Adding Custom Features

Edit `main.go` to add new endpoints:

```go
// Add custom API endpoint
app.Get("/api/custom", requireAuthOrRedirect, func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "message": "Custom endpoint",
    })
})
```

## Troubleshooting

### pgweb not loading

```bash
# Check pgweb container
docker logs pgweb

# Check connection
docker exec pgweb-proxy curl http://pgweb:8081
```

### Session issues

```bash
# Check session store
docker logs pgweb-proxy

# Restart services
docker-compose restart pgweb-proxy ota forwarder
```

### UI not displaying correctly

```bash
# Check if views are copied
docker exec pgweb-proxy ls -la /app/views

# Rebuild container
docker-compose build pgweb-proxy
```

## Comparison: Adminer vs pgweb

| Feature | Adminer | pgweb |
|---------|---------|-------|
| Database Support | Multi-DB | PostgreSQL only |
| UI Customization | Limited | Highly customizable |
| Performance | Good | Better for PostgreSQL |
| Size | ~500KB | ~10MB |
| Query History | No | Yes |
| Export Formats | SQL | CSV, JSON, XML |
| **Our Choice** | ❌ | ✅ |

## License

MIT License - See main project LICENSE file
