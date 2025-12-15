# 🐧 Linux Server Deployment Guide

Panduan deployment untuk Linux server (Ubuntu/Debian/CentOS).

## 📋 Prerequisites

### 1. Install Docker & Docker Compose

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group (no need sudo)
sudo usermod -aG docker $USER

# Logout and login again, or run:
newgrp docker

# Install Docker Compose (latest)
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker --version
docker-compose --version
```

### 2. Clone/Upload Project

```bash
# Option 1: Clone from git
git clone <your-repo-url> /opt/iot-apps
cd /opt/iot-apps

# Option 2: Upload via SCP
# From local PC:
scp -r d:\iot\apps user@server:/opt/iot-apps
```

## 🚀 Quick Start

### 1. Setup Environment

```bash
cd /opt/iot-apps

# Copy environment template
cp docs/ENV-TEMPLATE.md .env

# Edit configuration
nano .env
```

**Edit minimal:**

- `DB_PASSWORD` - Ganti dari default!
- `MQTT_PASSWORD` - Sesuaikan

### 2. Make Script Executable

```bash
# Give execute permission
chmod +x dev/deploy-helper.sh
```

### 3. Deploy!

```bash
# Start all services
./dev/deploy-helper.sh start
```

## 🔧 Deploy Helper Commands

### Basic Usage

```bash
# Show help
./dev/deploy-helper.sh help

# Start system (first time)
./dev/deploy-helper.sh start

# Smart update (preserve data)
./dev/deploy-helper.sh update

# Check status
./dev/deploy-helper.sh status

# View logs
./dev/deploy-helper.sh logs
```

### Service Management

```bash
# Rebuild single service
./dev/deploy-helper.sh rebuild forming
./dev/deploy-helper.sh rebuild backend
./dev/deploy-helper.sh rebuild ota-app

# Restart service
./dev/deploy-helper.sh update-svc forming

# Stop all
./dev/deploy-helper.sh stop

# Restart all
./dev/deploy-helper.sh restart
```

### Maintenance

```bash
# Backup database
./dev/deploy-helper.sh backup

# Clean containers (keep data)
./dev/deploy-helper.sh clean
```

## 📊 Service Access

After deployment, access services at:

- **Forming**: `http://your-server-ip:3000`
- **Forwarder**: `http://your-server-ip:8888/forwarder`
- **OTA**: `http://your-server-ip:9999`
- **EMQX**: `http://your-server-ip:18083` (admin/public)

## 🔒 Security Setup

### 1. Firewall Configuration

```bash
# Install UFW
sudo apt install ufw

# Allow SSH (important!)
sudo ufw allow ssh
sudo ufw allow 22/tcp

# Allow service ports
sudo ufw allow 3000/tcp   # Forming
sudo ufw allow 8888/tcp   # Forwarder
sudo ufw allow 9999/tcp   # OTA
sudo ufw allow 1883/tcp   # MQTT
sudo ufw allow 18083/tcp  # EMQX Dashboard

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

### 2. Secure EMQX Dashboard

```bash
# After first start, change EMQX admin password
# Access: http://your-ip:18083
# Login: admin/public
# Change password di dashboard!
```

### 3. Setup Reverse Proxy (Optional - Recommended)

```bash
# Install Nginx
sudo apt install nginx

# Create config
sudo nano /etc/nginx/sites-available/iot-dashboard
```

**Nginx config example:**

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /forwarder {
        proxy_pass http://localhost:8888/forwarder;
        proxy_set_header Host $host;
    }

    location /ota {
        proxy_pass http://localhost:9999;
        proxy_set_header Host $host;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/iot-dashboard /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 4. SSL with Let's Encrypt (Recommended)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal (already configured)
sudo certbot renew --dry-run
```

## 🔄 Common Workflows

### Daily Update (Code Changes)

```bash
# Pull latest code
git pull

# Smart update (preserve data)
./dev/deploy-helper.sh update

# Check logs
./dev/deploy-helper.sh logs
```

### Environment Changes

```bash
# Edit .env
nano .env

# Reload services
./dev/deploy-helper.sh update-svc forming
./dev/deploy-helper.sh update-svc backend
```

### Backup Routine

```bash
# Manual backup
./dev/deploy-helper.sh backup

# Setup cron for auto backup (daily at 2 AM)
crontab -e

# Add this line:
0 2 * * * cd /opt/iot-apps && ./dev/deploy-helper.sh backup >> /var/log/iot-backup.log 2>&1
```

### Monitor Resources

```bash
# Check Docker stats
docker stats

# Check disk usage
df -h

# Check logs size
du -sh /var/lib/docker/volumes/
```

## 🐛 Troubleshooting

### Port Already in Use

```bash
# Check what's using port 3000
sudo lsof -i :3000
sudo netstat -tulpn | grep 3000

# Kill process
sudo kill -9 <PID>
```

### Docker Permission Denied

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login, or:
newgrp docker
```

### Container Not Starting

```bash
# Check logs
docker-compose logs forming

# Check if container exists
docker ps -a

# Remove and restart
docker-compose down
./dev/deploy-helper.sh start
```

### Low Disk Space

```bash
# Clean unused images
docker system prune -a

# Clean unused volumes (⚠️ careful!)
docker volume prune
```

## 📈 Performance Tuning

### 1. Docker Logging

Edit `docker-compose.yml`:

```yaml
services:
  forming:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 2. System Resources

```bash
# Increase file descriptors
sudo nano /etc/security/limits.conf

# Add:
* soft nofile 65535
* hard nofile 65535

# Reboot
sudo reboot
```

### 3. PostgreSQL Optimization

```bash
# Edit PostgreSQL config in docker-compose.yml
# Add environment variables:
POSTGRES_SHARED_BUFFERS=256MB
POSTGRES_EFFECTIVE_CACHE_SIZE=1GB
```

## 🔄 Auto-Start on Boot

```bash
# Enable Docker service
sudo systemctl enable docker

# Docker Compose containers will auto-restart
# (already configured with restart: unless-stopped)

# Verify
docker-compose ps
```

## 📊 Monitoring Setup

### Install Portainer (Optional)

```bash
# Run Portainer
docker volume create portainer_data

docker run -d -p 9000:9000 \
  --name portainer \
  --restart=always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v portainer_data:/data \
  portainer/portainer-ce:latest

# Access: http://your-ip:9000
```

### System Monitoring

```bash
# Install htop
sudo apt install htop

# Monitor
htop

# Install docker stats dashboard
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

## 🆘 Support & Logs

### View Logs

```bash
# All services
./dev/deploy-helper.sh logs

# Specific service
docker-compose logs -f forming

# Last 100 lines
docker-compose logs --tail=100 forming

# Save logs
docker-compose logs > system-logs.txt
```

### Get System Info

```bash
# Docker info
docker info

# Container info
docker inspect forming-app

# Network info
docker network inspect servfor_iot-net
```

## 📝 Checklist

- [ ] Docker & Docker Compose installed
- [ ] Project uploaded/cloned to server
- [ ] `.env` configured
- [ ] `deploy-helper.sh` executable
- [ ] Firewall configured
- [ ] Services started
- [ ] EMQX admin password changed
- [ ] Backup cron configured
- [ ] SSL certificate installed (optional)
- [ ] Monitoring setup (optional)

---

**Ready to deploy!** 🚀

**Next**: `./dev/deploy-helper.sh start`
