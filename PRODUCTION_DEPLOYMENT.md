# Production Deployment Guide

## 📋 Overview

This guide explains how to deploy the IoT OTA system to production using pre-built Docker images from Docker Hub.

## 🎯 Differences: Development vs Production

| Aspect | Development | Production |
|--------|-------------|------------|
| **Docker Compose** | `docker-compose.yml` | `docker-compose.prod.yml` |
| **Images** | Built locally | Pull from Docker Hub |
| **Build Time** | 5-10 minutes | 1-2 minutes (pull only) |
| **Updates** | Rebuild required | Pull new image |
| **File** | Includes `build:` sections | No `build:` sections |

## 📁 Files

### Development
- **File:** `docker-compose.yml`
- **Purpose:** Local development
- **Images:** Built from source code
- **Usage:** `docker-compose up -d`

### Production
- **File:** `docker-compose.prod.yml`
- **Purpose:** Production deployment
- **Images:** Pre-built from Docker Hub
- **Usage:** `docker-compose -f docker-compose.prod.yml up -d`

## 🚀 Production Deployment Steps

### Prerequisites

1. **Docker & Docker Compose installed**
   ```bash
   docker --version
   docker-compose --version
   ```

2. **Environment file (.env)**
   ```bash
   # Copy example
   cp .env.example .env
   
   # Edit with your values
   nano .env
   ```

3. **Create uploads directory**
   ```bash
   mkdir -p ota/static/uploads
   ```

---

### Step 1: Pull Images from Docker Hub

```bash
# Pull all images
docker-compose -f docker-compose.prod.yml pull
```

This will download:
- `rennnagge/ota-app:latest`
- `rennnagge/forwarder-app:latest`
- `rennnagge/forming-app:latest`
- `sosedoff/pgweb`
- `postgres:13`
- `emqx:5.5`

---

### Step 2: Start Services

```bash
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps
```

Expected output:
```
NAME          STATUS         PORTS
postgres_db   Up (healthy)   5432/tcp
emqx          Up (healthy)   0.0.0.0:1883->1883/tcp, ...
ota-app       Up             0.0.0.0:9999->9999/tcp
forwarder     Up             0.0.0.0:8888->8000/tcp
forming-app   Up             0.0.0.0:3000->3000/tcp
pgweb         Up             0.0.0.0:8080->8081/tcp
```

---

### Step 3: Verify Services

```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs -f

# Check specific service
docker-compose -f docker-compose.prod.yml logs ota

# Test endpoints
curl http://localhost:9999/login
curl http://localhost:8888/login
curl http://localhost:3000
curl http://localhost:8080
```

---

## 🔄 Update to New Version

### Option 1: Update to Latest

```bash
# Pull latest images
docker-compose -f docker-compose.prod.yml pull

# Restart services
docker-compose -f docker-compose.prod.yml up -d
```

### Option 2: Update to Specific Version

Edit `docker-compose.prod.yml`:
```yaml
services:
  ota:
    image: rennnagge/ota-app:v1.0.0  # Specify version
```

Then:
```bash
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

---

## 🛠️ Common Operations

### View Logs
```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f ota

# Last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100
```

### Restart Service
```bash
# Restart specific service
docker-compose -f docker-compose.prod.yml restart ota

# Restart all
docker-compose -f docker-compose.prod.yml restart
```

### Stop Services
```bash
# Stop all
docker-compose -f docker-compose.prod.yml stop

# Stop specific
docker-compose -f docker-compose.prod.yml stop ota
```

### Remove Services
```bash
# Stop and remove containers
docker-compose -f docker-compose.prod.yml down

# Remove with volumes (WARNING: deletes data!)
docker-compose -f docker-compose.prod.yml down -v
```

---

## 📊 Service Ports

| Service | Port | Access |
|---------|------|--------|
| **OTA** | 9999 | http://server-ip:9999 |
| **Forwarder** | 8888 | http://server-ip:8888 |
| **Forming** | 3000 | http://server-ip:3000 |
| **pgweb** | 8080 | http://server-ip:8080 |
| **EMQX Dashboard** | 18083 | http://server-ip:18083 |
| **MQTT** | 1883 | mqtt://server-ip:1883 |

---

## 🔐 Security Recommendations

### 1. Use Firewall
```bash
# Allow only necessary ports
ufw allow 9999/tcp  # OTA
ufw allow 8888/tcp  # Forwarder
ufw allow 3000/tcp  # Forming
ufw allow 1883/tcp  # MQTT
ufw enable
```

### 2. Use Reverse Proxy (Nginx)
```nginx
server {
    listen 80;
    server_name ota.example.com;
    
    location / {
        proxy_pass http://localhost:9999;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 3. Use SSL/TLS
```bash
# Install certbot
apt install certbot python3-certbot-nginx

# Get certificate
certbot --nginx -d ota.example.com
```

### 4. Change Default Passwords
Edit `.env`:
```env
DB_PASSWORD=your_strong_password_here
MQTT_PASSWORD=your_mqtt_password_here
```

---

## 📦 Backup & Restore

### Backup Database
```bash
# Backup PostgreSQL
docker exec postgres_db pg_dump -U postgres servfi > backup.sql

# Or use docker-compose
docker-compose -f docker-compose.prod.yml exec postgres pg_dump -U postgres servfi > backup.sql
```

### Restore Database
```bash
# Restore
docker exec -i postgres_db psql -U postgres servfi < backup.sql

# Or use docker-compose
docker-compose -f docker-compose.prod.yml exec -T postgres psql -U postgres servfi < backup.sql
```

### Backup Volumes
```bash
# Backup all volumes
docker run --rm \
  -v servfor_postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres_data_backup.tar.gz /data
```

---

## 🔍 Troubleshooting

### Service Won't Start
```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs service_name

# Check container status
docker-compose -f docker-compose.prod.yml ps

# Restart service
docker-compose -f docker-compose.prod.yml restart service_name
```

### Can't Pull Images
```bash
# Check Docker Hub login
docker login

# Manually pull
docker pull rennnagge/ota-app:latest

# Check internet connection
ping hub.docker.com
```

### Database Connection Error
```bash
# Check database is healthy
docker-compose -f docker-compose.prod.yml ps postgres

# Check environment variables
docker-compose -f docker-compose.prod.yml config

# Restart database
docker-compose -f docker-compose.prod.yml restart postgres
```

---

## 📈 Monitoring

### Check Resource Usage
```bash
# All containers
docker stats

# Specific container
docker stats ota-app
```

### Health Checks
```bash
# Check health status
docker-compose -f docker-compose.prod.yml ps

# Inspect health
docker inspect --format='{{.State.Health.Status}}' ota-app
```

---

## 🎯 Production Checklist

- [ ] Environment variables configured in `.env`
- [ ] Images pulled from Docker Hub
- [ ] Services started and healthy
- [ ] Firewall configured
- [ ] SSL/TLS certificates installed
- [ ] Backup strategy in place
- [ ] Monitoring setup
- [ ] Logs rotation configured
- [ ] Default passwords changed
- [ ] Documentation reviewed

---

## 📝 Quick Reference

### Start Production
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Update Production
```bash
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

### View Logs
```bash
docker-compose -f docker-compose.prod.yml logs -f
```

### Stop Production
```bash
docker-compose -f docker-compose.prod.yml down
```

### Backup Database
```bash
docker-compose -f docker-compose.prod.yml exec postgres pg_dump -U postgres servfi > backup.sql
```

---

**Last Updated:** 2025-12-17  
**Version:** 1.0  
**Status:** ✅ Production Ready
