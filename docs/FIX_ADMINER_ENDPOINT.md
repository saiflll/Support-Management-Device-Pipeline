# Fix: Cannot GET /adminer

## ✅ Masalah Sudah Diperbaiki

### Penyebab:
- Endpoint sudah diganti dari `/adminer` → `/database`
- Tapi HTML masih menggunakan link lama `/adminer`

### Yang Sudah Diperbaiki:

1. **OTA Dashboard** (`ota/views/index.html`)
   ```html
   <!-- Before -->
   <a href="/adminer" target="_blank">
   
   <!-- After -->
   <a href="/database" target="_blank">
   ```

2. **Forwarder Dashboard** (`forward/internal/forwarder/index.html`)
   ```html
   <!-- Before -->
   <a href="/adminer" target="_blank">
   
   <!-- After -->
   <a href="/database" target="_blank">
   ```

3. **Backend Endpoints**
   - ✅ OTA: `GET /database` → redirect ke pgweb-proxy
   - ✅ Forwarder: `GET /database` → redirect ke pgweb-proxy

## 🔄 Cara Menggunakan Sekarang

### 1. Restart Services
```bash
docker-compose restart ota backend
```

### 2. Clear Browser Cache
- Tekan `Ctrl + Shift + R` (hard refresh)
- Atau clear cache di browser settings

### 3. Test Akses Database

#### Via OTA:
1. Login ke http://localhost:9999
2. Klik tombol **"Database"** di header
3. ✅ Akan redirect ke pgweb dengan custom UI

#### Via Forwarder:
1. Login ke http://localhost:8888
2. Klik tombol **"Database"** di header
3. ✅ Akan redirect ke pgweb dengan custom UI

## 🔍 Verifikasi

### Test Endpoint:
```bash
# Test OTA database endpoint
curl -I http://localhost:9999/database
# Should return: 302 Found (redirect)

# Test Forwarder database endpoint
curl -I http://localhost:8888/database
# Should return: 302 Found (redirect)

# Old endpoint should NOT work
curl -I http://localhost:9999/adminer
# Should return: 404 Not Found
```

## 📝 Checklist

- [x] Update OTA HTML: `/adminer` → `/database`
- [x] Update Forwarder HTML: `/adminer` → `/database`
- [x] Verify no more `/adminer` references
- [x] Backend endpoints already using `/database`
- [x] pgweb-proxy ready to receive requests

## 🎯 Summary

**Endpoint Baru:**
- ✅ `http://localhost:9999/database` (OTA)
- ✅ `http://localhost:8888/database` (Forwarder)

**Endpoint Lama (Tidak Digunakan Lagi):**
- ❌ `http://localhost:9999/adminer` (404)
- ❌ `http://localhost:8888/adminer` (404)

**Akses Database:**
- Klik tombol "Database" di dashboard
- Akan membuka custom UI dengan pgweb embedded
- Tema matching dengan OTA dashboard

---

**Status:** ✅ Fixed  
**Date:** 2025-12-17
