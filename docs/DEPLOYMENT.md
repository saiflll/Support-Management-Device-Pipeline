# 🚀 Deployment Guide - Debian Linux

## 📦 Prerequisites di Server Debian

Pastikan sudah terinstal:

- [x] Docker
- [x] Docker Compose
- [x] EMQX container (dengan network `servfor_iot-net`)
- [x] PostgreSQL container `postgres_db`

---

## 🔑 Setup MQTT User di EMQX (Penting!)

Sebelum deploy, pastikan user MQTT sudah dibuat di EMQX:

```bash
# Login ke EMQX container
docker exec -it emqx sh

# Buat user MQTT
emqx_ctl users add forming_app forming_secure_2025

# Atau via EMQX Dashboard:
# http://<server-ip>:18083
# Username: admin
# Password: public
# Menu: Authentication -> Password-Based -> Add User
```

**ATAU jika EMQX tidak pakai authentication**, edit `docker-compose.yml`:

```yaml
# Comment out MQTT credentials:
# - MQTT_USER=forming_app
# - MQTT_PASSWORD=forming_secure_2025
```

---

## 📤 Upload ke Server Debian

### **Dari Windows ke Debian:**

```powershell
# Dari folder forming di Windows
# Upload semua file kecuali .env.local dan forming.exe
scp -r Dockerfile docker-compose.yml go.mod go.sum main.go skip_log.go deploy.sh views/ public/ app@172.20.100.11:~/apps/forming/
```

**ATAU upload satu per satu:**

```powershell
cd "d:\iot\suhu ck 3\apps\forming"

# Upload files
scp Dockerfile app@172.20.100.11:~/apps/forming/
scp docker-compose.yml app@172.20.100.11:~/apps/forming/
scp deploy.sh app@172.20.100.11:~/apps/forming/
scp *.go app@172.20.100.11:~/apps/forming/
scp go.mod go.sum app@172.20.100.11:~/apps/forming/

# Upload folders
scp -r views app@172.20.100.11:~/apps/forming/
scp -r public app@172.20.100.11:~/apps/forming/
```

---

## 🚀 Deploy di Server Debian

### **Step 1: SSH ke Server**

```bash
ssh app@172.20.100.11
```

### **Step 2: Masuk ke Folder**

```bash
cd ~/apps/forming
```

### **Step 3: Beri Permission Execute**

```bash
chmod +x deploy.sh
```

### **Step 4: (Optional) Edit Credentials**

```bash
# Edit password jika perlu
nano docker-compose.yml

# Ubah:
# - DB_PASSWORD=password_production_anda
# - MQTT_PASSWORD=password_mqtt_anda
```

### **Step 5: Jalankan Deployment**

```bash
./deploy.sh
```

**ATAU manual:**

```bash
# Stop existing
docker-compose down

# Build image baru
docker-compose build --no-cache

# Start container
docker-compose up -d

# Monitor logs
docker logs forming-app -f
```

---

## ✅ Verifikasi Deployment

### **Check Container Status:**

```bash
docker ps | grep forming
```

Expected output:

```
forming-app   Up X minutes   0.0.0.0:3000->3000/tcp
```

### **Check Logs:**

```bash
docker logs forming-app --tail 50
```

Expected output:

```
✅ Connected to PostgreSQL
✅ Table 'production_mdcw' ensured
✅ Table 'skip_log' ensured
✅ MQTT Connected successfully!
✅ Subscribed to topic: production/mdcw
✅ Fiber v2.52.10 running on port 3000
```

### **Check Access:**

```bash
# Dari server
curl http://localhost:3000

# Dari browser
http://172.20.100.11:3000
```

---

## 🔍 Troubleshooting

### **MQTT Error: "bad user name or password"**

**Solusi 1:** Buat user di EMQX

```bash
docker exec -it emqx sh
emqx_ctl users add forming_app forming_secure_2025
exit
docker-compose restart
```

**Solusi 2:** Disable authentication

```bash
nano docker-compose.yml
# Comment out:
# - MQTT_USER=forming_app
# - MQTT_PASSWORD=forming_secure_2025

docker-compose restart
```

### **404 Error - JS Files Not Found**

Pastikan Dockerfile sudah di-update (baris 26-27):

```dockerfile
COPY --from=builder /app/public ./public
```

Lalu rebuild:

```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### **Cannot Connect to Database**

Check network:

```bash
docker network ls | grep servfor
docker network inspect servfor_iot-net

# Pastikan postgres_db ada di network yang sama
```

Fix:

```bash
# Buat network jika belum ada
docker network create servfor_iot-net

# Atau edit docker-compose.yml, ganti dengan network yang benar
```

### **Port 3000 Already in Use**

Edit docker-compose.yml:

```yaml
ports:
  - "3001:3000" # Ganti external port
```

---

## 📊 Monitor Production

### **Real-time Logs:**

```bash
docker logs forming-app -f
```

### **Last 100 Lines:**

```bash
docker logs forming-app --tail 100
```

### **Container Stats:**

```bash
docker stats forming-app
```

### **Restart Container:**

```bash
docker-compose restart
```

### **Stop Container:**

```bash
docker-compose down
```

---

## 🔄 Update/Redeploy

Jika ada perubahan code:

```bash
# Upload file baru dari Windows
scp main.go app@172.20.100.11:~/apps/forming/

# SSH ke server
ssh app@172.20.100.11
cd ~/apps/forming

# Redeploy
./deploy.sh
```

---

## 📝 Quick Commands Cheatsheet

```bash
# Deploy
./deploy.sh

# View logs
docker logs forming-app -f

# Restart
docker-compose restart

# Stop
docker-compose down

# Start (no rebuild)
docker-compose up -d

# Rebuild & Start
docker-compose down && docker-compose build --no-cache && docker-compose up -d

# Check status
docker ps | grep forming
curl http://localhost:3000
```

---

## ✅ Deployment Checklist

- [ ] EMQX user created (`forming_app`)
- [ ] PostgreSQL container running (`postgres_db`)
- [ ] Network exists (`servfor_iot-net`)
- [ ] Files uploaded to `~/apps/forming/`
- [ ] `deploy.sh` has execute permission
- [ ] Credentials configured in `docker-compose.yml`
- [ ] `./deploy.sh` executed successfully
- [ ] Logs show "Connected to PostgreSQL"
- [ ] Logs show "MQTT Connected successfully"
- [ ] Dashboard accessible at `http://<server-ip>:3000`
- [ ] JS files loading (no 404 errors)

---

**Good luck with deployment! 🚀**
