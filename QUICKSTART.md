# 🚀 Quick Start Guide

## Persiapan (First Time Setup)

### 1. Install Prerequisites
- Docker Desktop (Windows/Mac) atau Docker Engine (Linux)
- Docker Compose

### 2. Setup Environment Variables
Buat file `.env` di root directory:
```env
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here

# MQTT Configuration
MQTT_BROKER=tcp://172.20.100.11:1883
PUBLIC_BROKER=tcp://broker.emqx.io:1883
```

### 3. Get Telegram Bot Token
1. Buka [@BotFather](https://t.me/BotFather) di Telegram
2. Send `/newbot`
3. Follow instructions
4. Copy token yang diberikan

### 4. Get Telegram Chat ID
1. Buka [@userinfobot](https://t.me/userinfobot) di Telegram
2. Start bot
3. Copy Chat ID Anda

## 🎯 Deploy (First Time)

### Windows:
```cmd
deploy.bat
```

### Linux/Mac:
```bash
chmod +x deploy.sh
./deploy.sh
```

### Manual (All Platforms):
```bash
docker-compose up --build -d
```

## 🌐 Access Dashboard

1. Open browser: http://localhost:9999
2. Click "Kirim Kode ke Telegram"
3. Check Telegram for verification code
4. Enter 6-digit code
5. Done! ✅

## 📊 Features Overview

### OTA Management
- Upload firmware files (.bin)
- Manage firmware versions
- Rename/Delete files
- Copy download links

### Node Monitoring
- Real-time node status
- Online/Offline detection
- Auto-refresh every 5 seconds
- Node logs viewer

### Forwarder Monitor
- Buffer status
- Forward countdown
- Buffered data preview
- Forward success/fail status

## 🔄 Update Application

```bash
# Pull latest code (if using git)
git pull

# Rebuild and restart
docker-compose up --build -d
```

## 🐛 Common Issues

### Issue: Port 9999 already in use
**Solution 1**: Change port in `docker-compose.yml`:
```yaml
ports:
  - "8080:9999"  # Use port 8080 instead
```

**Solution 2**: Kill process using port 9999:
```bash
# Windows
netstat -ano | findstr :9999
taskkill /PID <PID> /F

# Linux
sudo lsof -t -i:9999 | xargs sudo kill -9
```

### Issue: Session error / Cannot login
**Check**: Environment variables are set correctly
```bash
# View current env
docker-compose config

# Restart services
docker-compose restart
```

### Issue: MQTT not connecting
**Check**: 
1. MQTT broker is running and accessible
2. Firewall allows MQTT port (1883)
3. Network connectivity

```bash
# Test MQTT connection
docker-compose logs forwarder | grep "MQTT"
```

### Issue: Container won't start
```bash
# Check logs
docker-compose logs

# Force rebuild
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## 📱 Mobile Access

To access from mobile/other devices on same network:

1. Find server IP:
```bash
# Windows
ipconfig

# Linux/Mac
ifconfig
```

2. Access via: `http://<SERVER_IP>:9999`

3. **Important**: Make sure firewall allows port 9999

## 🔒 Production Deployment

### 1. Use HTTPS (Recommended)
Add nginx reverse proxy with SSL:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:9999;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 2. Use Strong Passwords
Change default credentials if any

### 3. Enable Rate Limiting
Add nginx rate limiting:
```nginx
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

location /login {
    limit_req zone=login burst=3;
    proxy_pass http://localhost:9999;
}
```

### 4. Regular Backups
```bash
# Backup uploaded files
tar -czf ota-backup-$(date +%Y%m%d).tar.gz ./ota/static/uploads/

# Backup to remote server
scp ota-backup-*.tar.gz user@backup-server:/path/to/backups/
```

## 📈 Monitoring

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f ota
docker-compose logs -f forwarder

# Last 100 lines
docker-compose logs --tail=100
```

### Check Resources
```bash
# Container stats
docker stats

# Disk usage
docker system df
```

## 🎨 Customization

### Change Theme Colors
Edit `ota/views/index.html` and `login.html`:
```css
/* Change primary color (Purple) */
.text-glow-purple { color: #your-color; }

/* Change secondary color (Cyan) */
.text-glow-cyan { color: #your-color; }
```

### Change Auto-refresh Interval
Edit `ota/views/index.html`:
```javascript
// Default: 5000ms (5 seconds)
setInterval(updateForwarderStatus, 5000);
```

## 🆘 Need Help?

1. Check `DEPLOYMENT.md` for detailed guide
2. Check Docker logs: `docker-compose logs`
3. Check application logs in container
4. Verify environment variables
5. Test network connectivity

## 📞 Support Commands

```bash
# Full restart
docker-compose down && docker-compose up -d

# Clean everything and start fresh
docker-compose down -v --rmi all
docker-compose up --build -d

# Export logs to file
docker-compose logs > logs.txt
```

---

✨ **Pro Tip**: Bookmark `http://localhost:9999` setelah pertama kali login agar lebih cepat akses!
