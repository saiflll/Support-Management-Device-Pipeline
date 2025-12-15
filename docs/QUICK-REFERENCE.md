# ⚡ Quick Reference - IoT System

Cheat sheet untuk command yang sering digunakan.

## 🚀 Deployment

```powershell
# Fresh deployment (first time)
.\dev\deploy-helper.ps1 start

# Smart update (preserve data - RECOMMENDED!)
.\dev\deploy-helper.ps1 update

# Rebuild single service
.\dev\deploy-helper.ps1 rebuild forming
.\dev\deploy-helper.ps1 rebuild backend
.\dev\deploy-helper.ps1 rebuild ota-app

# Restart service (no rebuild)
.\dev\deploy-helper.ps1 update-svc forming

# Stop/Restart all
.\dev\deploy-helper.ps1 stop
.\dev\deploy-helper.ps1 restart

# Status + health check
.\dev\deploy-helper.ps1 status
```

## 📊 Monitoring

```bash
# View logs (all services)
docker-compose logs -f

# View logs (specific service)
docker-compose logs -f forming
docker-compose logs -f backend
docker-compose logs -f ota

# Last 50 lines
docker-compose logs --tail=50 forming

# Container status
docker-compose ps

# Resource usage
docker stats
```

## 🔧 Service Management

```bash
# Restart service tertentu
docker-compose restart forming

# Rebuild service
docker-compose build forming
docker-compose up -d forming

# Stop service tertentu
docker-compose stop forming

# Start service tertentu
docker-compose start forming
```

## 💾 Database

```bash
# Backup database
.\deploy-helper.ps1 backup

# Manual backup
docker exec postgres_db pg_dump -U postgres servfi > backup.sql

# Restore database
docker exec -i postgres_db psql -U postgres servfi < backup.sql

# Connect ke database
docker exec -it postgres_db psql -U postgres -d servfi

# Query dari command line
docker exec postgres_db psql -U postgres -d servfi -c "SELECT COUNT(*) FROM records;"
```

## 🌐 Access URLs

| Service   | URL                             | Credentials  |
| --------- | ------------------------------- | ------------ |
| Forming   | http://localhost:3000           | -            |
| Forwarder | http://localhost:8888/forwarder | -            |
| OTA       | http://localhost:9999           | -            |
| EMQX      | http://localhost:18083          | admin/public |

## 🔍 Debug

```bash
# Cek network
docker network ls
docker network inspect servfor_iot-net

# Exec into container
docker exec -it forming-app sh
docker exec -it postgres_db bash

# Health check manual
docker exec postgres_db pg_isready -U postgres
docker exec emqx emqx ping

# Port checking
netstat -ano | findstr "3000"
netstat -ano | findstr "8888"
netstat -ano | findstr "9999"
```

## 📡 MQTT Testing

```bash
# Via EMQX Dashboard
# 1. Open http://localhost:18083
# 2. Login: admin/public
# 3. Menu: Tools > Websocket
# 4. Connect
# 5. Subscribe: production/mdcw
# 6. Publish test message

# Test publish JSON
# Topic: production/mdcw
# Payload:
{
  "ts": "2025-12-15 08:45:30",
  "reg2": 1,
  "reg5": 250,
  "reg114": 1,
  "prefix": "A1"
}
```

## 🧹 Cleanup

```bash
# Stop dan remove containers (data tetap)
docker-compose down

# Remove containers + volumes (⚠️ DATA HILANG!)
docker-compose down -v

# Clean unused images
docker image prune -a

# Clean system (all unused)
docker system prune -a
```

## 🔑 MQTT User Management

```bash
# Via EMQX Dashboard
# 1. http://localhost:18083
# 2. Access Control > Authentication > Password-Based
# 3. Add user: servfi_app / S3cr3tP@ssw0rd!
```

## 🔄 Update Workflow

```bash
# 1. Pull changes
git pull

# 2. Rebuild
docker-compose build

# 3. Restart
docker-compose up -d

# 4. Verify
docker-compose ps
docker-compose logs -f
```

## 🐛 Common Issues

### Service tidak start

```bash
docker-compose ps
docker-compose logs [service_name]
```

### Port conflict

```bash
# Cek port usage
netstat -ano | findstr "[port]"

# Ubah port di docker-compose.yml
# "3001:3000" instead of "3000:3000"
```

### MQTT auth failed

```bash
# Check EMQX logs
docker-compose logs emqx

# Verify credentials di EMQX Dashboard
# Pastikan user "servfi_app" ada
```

### Database connection failed

```bash
# Check PostgreSQL
docker-compose logs postgres

# Check healthcheck
docker inspect postgres_db | grep Health

# Restart
docker-compose restart postgres
```

## 📞 Help

```powershell
# Helper script help
.\deploy-helper.ps1 help

# Docker compose help
docker-compose --help

# View README
cat README.md

# Full guide
cat INTEGRATION-GUIDE.md
```

---

💡 **Tip**: Simpan file ini untuk referensi cepat!
