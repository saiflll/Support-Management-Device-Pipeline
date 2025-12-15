# 🚀 Deployment Strategy Guide

Panduan lengkap untuk deploy dan update sistem dengan aman tanpa data loss.

## 📋 Deployment Scenarios

### 1️⃣ Fresh Deployment (First Time)

**Use Case**: Deploy sistem pertama kali

```powershell
.\dev\deploy-helper.ps1 start
```

**What it does:**

- ✅ Build semua services (postgres, emqx, forming, forwarder, ota)
- ✅ Create volumes baru untuk data
- ✅ Setup network
- ✅ Start semua containers

---

### 2️⃣ Smart Update (Recommended)

**Use Case**: Update code aplikasi **TANPA** rebuild database/MQTT

```powershell
.\dev\deploy-helper.ps1 update
```

**What it does:**

- ✅ Rebuild: `forming`, `forwarder`, `ota` (app services)
- ❌ Skip rebuild: `postgres`, `emqx` (data services)
- ✅ **Data PRESERVED** - Database & MQTT config tetap aman!
- ✅ Restart hanya app services

**Perfect for:**

- Update code forming/forwarder/ota
- Deploy fitur baru
- Bug fixes
- Daily development

---

### 3️⃣ Rebuild Single Service

**Use Case**: Rebuild hanya 1 service yang berubah

```powershell
# Rebuild forming saja
.\dev\deploy-helper.ps1 rebuild forming

# Rebuild forwarder saja
.\dev\deploy-helper.ps1 rebuild backend

# Rebuild OTA saja
.\dev\deploy-helper.ps1 rebuild ota-app
```

**What it does:**

- ✅ Rebuild image service yang dipilih
- ✅ Restart service tersebut
- ✅ Service lain tetap running
- ✅ **No data loss**

**Perfect for:**

- Perubahan kecil di 1 service
- Testing service tertentu
- Quick iteration

---

### 4️⃣ Update Without Rebuild

**Use Case**: Restart service tanpa rebuild (config change only)

```powershell
# Restart forming (load .env baru)
.\dev\deploy-helper.ps1 update-svc forming

# Restart forwarder
.\dev\deploy-helper.ps1 update-svc backend
```

**What it does:**

- ✅ Restart service (reload environment variables)
- ❌ No rebuild
- ✅ Super cepat!

**Perfect for:**

- Update environment variables (.env)
- Reload configuration
- Quick service restart

---

## 🎯 Decision Tree

```
┌─ Ada perubahan code? ─┐
│                        │
YES                      NO
│                        │
├─ Semua service?        └─> update-svc (restart only)
│  │
│  YES ──> update (smart)
│  NO  ──> rebuild [service]
│
└─ First time? ──> start (fresh)
```

## 📊 Comparison Table

| Command            | PostgreSQL | EMQX     | Apps         | Build Time | Data Safe   |
| ------------------ | ---------- | -------- | ------------ | ---------- | ----------- |
| `start`            | ✅ Build   | ✅ Build | ✅ Build     | ~5 min     | ⚠️ New data |
| `update`           | ❌ Skip    | ❌ Skip  | ✅ Build     | ~2 min     | ✅ Safe     |
| `rebuild [svc]`    | ❌ Skip    | ❌ Skip  | ✅ 1 service | ~30 sec    | ✅ Safe     |
| `update-svc [svc]` | ❌ Skip    | ❌ Skip  | ❌ Skip      | ~5 sec     | ✅ Safe     |

## 💡 Best Practices

### Daily Development

```powershell
# Edit code di forming/
# Test local
go run .

# Deploy ke server
.\dev\deploy-helper.ps1 rebuild forming
```

### Weekly Updates

```powershell
# Multiple services berubah
.\dev\deploy-helper.ps1 update
```

### Environment Changes Only

```powershell
# Edit .env
notepad .env

# Reload service
.\dev\deploy-helper.ps1 update-svc forming
.\dev\deploy-helper.ps1 update-svc backend
```

### Before Major Update

```powershell
# Backup dulu!
.\dev\deploy-helper.ps1 backup

# Then update
.\dev\deploy-helper.ps1 update
```

## 🔒 Data Safety

### Protected Data (NEVER rebuilt by 'update'):

- ✅ PostgreSQL database (`postgres_data` volume)
- ✅ EMQX configuration (`emqx_data` volume)
- ✅ EMQX logs (`emqx_log` volume)
- ✅ OTA firmware uploads (`./ota/static/uploads` bind mount)

### Rebuild Safe:

- ✅ `update` command
- ✅ `rebuild [service]` command
- ✅ `update-svc [service]` command

### ⚠️ Data Loss Risk:

- ❌ `docker-compose down -v` (removes volumes!)
- ❌ Manual volume deletion

## 🛠️ Common Workflows

### Scenario 1: Update Forming Code

```powershell
# 1. Edit code di forming/main.go atau forming/lib/
# 2. Test locally
cd forming
go run .

# 3. Deploy
cd ..
.\dev\deploy-helper.ps1 rebuild forming

# 4. Check logs
docker-compose logs -f forming
```

### Scenario 2: Update Multiple Services

```powershell
# Edit forming + forwarder code
# Deploy all
.\dev\deploy-helper.ps1 update

# Verify
.\dev\deploy-helper.ps1 status
```

### Scenario 3: Change Environment Config

```powershell
# Edit .env
notepad .env

# Reload affected services
.\dev\deploy-helper.ps1 update-svc forming
.\dev\deploy-helper.ps1 update-svc backend

# Verify
docker-compose logs -f forming backend
```

### Scenario 4: Emergency Rollback

```powershell
# 1. Stop problematic service
docker-compose stop forming

# 2. Restore previous image (if tagged)
docker tag forming-app:previous forming-app:latest

# 3. Start again
docker-compose up -d forming
```

## 📝 Command Reference

### Full Command List:

```powershell
# Deployment
.\dev\deploy-helper.ps1 start           # Fresh deployment
.\dev\deploy-helper.ps1 update          # Smart update (preserve data)

# Service Management
.\dev\deploy-helper.ps1 stop            # Stop all
.\dev\deploy-helper.ps1 restart         # Restart all
.\dev\deploy-helper.ps1 rebuild forming # Rebuild one service
.\dev\deploy-helper.ps1 update-svc forming # Restart one service

# Monitoring
.\dev\deploy-helper.ps1 logs            # View logs
.\dev\deploy-helper.ps1 status          # Check status

# Maintenance
.\dev\deploy-helper.ps1 backup          # Backup database
.\dev\deploy-helper.ps1 clean           # Remove containers
```

## ⚠️ Important Notes

1. **Always backup before major updates:**

   ```powershell
   .\dev\deploy-helper.ps1 backup
   ```

2. **'update' is your friend** - Use it for most deployments

   - Preserves data
   - Faster than 'start'
   - Safe for production

3. **Never use 'down -v' in production:**

   ```powershell
   # ❌ DANGER - This deletes all data!
   docker-compose down -v

   # ✅ SAFE - This keeps data
   docker-compose down
   ```

4. **Check logs after deployment:**
   ```powershell
   docker-compose logs -f forming backend ota-app
   ```

---

**Recommended Workflow**: `backup` → `update` → `status` → `logs`
