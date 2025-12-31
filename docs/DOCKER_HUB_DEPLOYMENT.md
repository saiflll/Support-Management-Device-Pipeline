# Docker Hub Deployment Guide

## 📦 Overview

Semua service sudah dioptimasi dengan **multi-stage build** untuk:
- ✅ **Smaller image size** - hanya runtime dependencies
- ✅ **Better security** - run as non-root user
- ✅ **Health checks** - automatic container monitoring
- ✅ **Production-ready** - optimized build flags

## 🏗️ Dockerfile Optimizations

### Common Features (All Services)

1. **Multi-stage Build**
   - Stage 1: Build (golang:alpine)
   - Stage 2: Runtime (alpine:3.18)

2. **Build Optimizations**
   ```dockerfile
   RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
       -ldflags='-w -s -extldflags "-static"' \
       -a -installsuffix cgo \
       -o binary main.go
   ```
   - `-w -s`: Strip debug info (smaller binary)
   - `-extldflags "-static"`: Static linking
   - `CGO_ENABLED=0`: No C dependencies

3. **Security**
   ```dockerfile
   RUN addgroup -g 1000 appuser && \
       adduser -D -u 1000 -G appuser appuser
   USER appuser
   ```
   - Run as non-root user
   - UID/GID 1000 for consistency

4. **Health Checks**
   ```dockerfile
   HEALTHCHECK --interval=30s --timeout=3s \
       CMD wget --spider http://localhost:PORT/ || exit 1
   ```

5. **Timezone**
   ```dockerfile
   ENV TZ=Asia/Jakarta
   ```

## 📊 Image Size Comparison

| Service | Before | After | Reduction |
|---------|--------|-------|-----------|
| OTA | ~400MB | ~20MB | 95% ⬇️ |
| Forwarder | ~400MB | ~20MB | 95% ⬇️ |
| Forming | ~400MB | ~20MB | 95% ⬇️ |
| pgweb Proxy | ~400MB | ~15MB | 96% ⬇️ |

## 🚀 Build & Push to Docker Hub

### Prerequisites

1. **Docker Hub Account**
   ```bash
   docker login
   ```

2. **Update Username**
   Edit script dan ganti `rennn` dengan username Anda:
   - `build-and-push.sh` (Linux/Mac)
   - `build-and-push.ps1` (Windows)

### Build All Services

#### Linux/Mac:
```bash
# Make script executable
chmod +x build-and-push.sh

# Build and push with version tag
./build-and-push.sh v1.0.0

# Or use latest
./build-and-push.sh
```

#### Windows (PowerShell):
```powershell
# Build and push with version tag
.\build-and-push.ps1 -Version v1.0.0

# Or use latest
.\build-and-push.ps1
```

### Build Individual Service

```bash
# OTA
docker build -t rennn/ota-app:latest ./ota
docker push rennn/ota-app:latest

# Forwarder
docker build -t rennn/forwarder-app:latest ./forward
docker push rennn/forwarder-app:latest

# Forming
docker build -t rennn/forming-app:latest ./forming
docker push rennn/forming-app:latest

# pgweb Proxy
docker build -t rennn/pgweb-proxy:latest ./pgweb
docker push rennn/pgweb-proxy:latest
```

## 📝 Docker Compose Configuration

### Using Docker Hub Images

Update `docker-compose.yml`:

```yaml
services:
  ota:
    image: rennn/ota-app:latest
    # Remove build section if using pre-built image
    # build:
    #   context: ./ota
    
  backend:
    image: rennn/forwarder-app:latest
    # Remove build section
    
  forming:
    image: rennn/forming-app:latest
    # Remove build section
    
  pgweb-proxy:
    image: rennn/pgweb-proxy:latest
    # Remove build section
```

### Using Specific Version

```yaml
services:
  ota:
    image: rennn/ota-app:v1.0.0
```

## 🔍 Verify Images

### Check Image Size

```bash
docker images | grep rennn
```

Expected output:
```
rennn/ota-app          latest    abc123    20MB
rennn/forwarder-app    latest    def456    20MB
rennn/forming-app      latest    ghi789    20MB
rennn/pgweb-proxy      latest    jkl012    15MB
```

### Test Image

```bash
# Pull from Docker Hub
docker pull rennn/ota-app:latest

# Run container
docker run -p 9999:9999 rennn/ota-app:latest

# Check health
docker ps
```

## 📋 Image Details

### OTA Service
```
Image: rennn/ota-app
Port: 9999
Health: /login
Size: ~20MB
```

### Forwarder Service
```
Image: rennn/forwarder-app
Port: 8888
Health: /login
Size: ~20MB
```

### Forming Service
```
Image: rennn/forming-app
Port: 3000
Health: /
Size: ~20MB
```

### pgweb Proxy
```
Image: rennn/pgweb-proxy
Port: 8080
Health: /health
Size: ~15MB
```

## 🔒 Security Best Practices

### 1. Use Specific Versions
```yaml
# ❌ Bad - always pulls latest
image: rennn/ota-app:latest

# ✅ Good - pinned version
image: rennn/ota-app:v1.0.0
```

### 2. Scan Images
```bash
# Using Docker Scout
docker scout cves rennn/ota-app:latest

# Using Trivy
trivy image rennn/ota-app:latest
```

### 3. Sign Images
```bash
# Enable Docker Content Trust
export DOCKER_CONTENT_TRUST=1
docker push rennn/ota-app:latest
```

## 🔄 Update Workflow

### 1. Make Changes
```bash
# Edit code
vim ota/main.go
```

### 2. Build & Test Locally
```bash
docker-compose build ota
docker-compose up ota
```

### 3. Push to Docker Hub
```bash
./build-and-push.sh v1.0.1
```

### 4. Update Production
```bash
# Update docker-compose.yml
image: rennn/ota-app:v1.0.1

# Pull and restart
docker-compose pull
docker-compose up -d
```

## 📊 Monitoring

### Check Container Health

```bash
# All containers
docker ps

# Specific service
docker inspect --format='{{.State.Health.Status}}' ota-app
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f ota
```

## 🐛 Troubleshooting

### Build Fails

```bash
# Clear build cache
docker builder prune

# Rebuild without cache
docker build --no-cache -t rennn/ota-app:latest ./ota
```

### Push Fails

```bash
# Re-login to Docker Hub
docker logout
docker login

# Check credentials
docker info | grep Username
```

### Image Too Large

```bash
# Check layers
docker history rennn/ota-app:latest

# Use dive to analyze
dive rennn/ota-app:latest
```

## 📚 Additional Resources

- [Docker Hub](https://hub.docker.com/)
- [Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Security Scanning](https://docs.docker.com/scout/)

## ✅ Checklist

- [ ] All Dockerfiles use multi-stage build
- [ ] All services run as non-root user
- [ ] Health checks configured
- [ ] .dockerignore files created
- [ ] Build script tested
- [ ] Images pushed to Docker Hub
- [ ] docker-compose.yml updated
- [ ] Production deployment tested

---

**Last Updated:** 2025-12-17  
**Version:** 1.0  
**Status:** ✅ Production Ready
