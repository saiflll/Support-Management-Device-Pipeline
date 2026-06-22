# Hybrid Automation Gateway Engine (HID + VCP)

Aplikasi gateway otomasi hibrida untuk mendengarkan masukan data dari pemindai USB HID (Keyboard Emulation) dan Virtual COM Port (VCP) secara simultan, lalu mengirimkannya ke broker MQTT.

## Fitur Utama

1. **HID Scanner Engine**: Menggunakan global keyboard hook dengan filter kecepatan semburan (*burst filter*). Karakter yang diketik lebih lambat dari `0.30 detik` akan otomatis diabaikan (karena dianggap ketikan manusia biasa, bukan scanner barcode/RFID).
2. **VCP Engine**: Secara otomatis mendeteksi dan mengunci stream port serial virtual (COM) yang terpasang pada komputer untuk dialirkan datanya secara real-time.
3. **MQTT Gateway**: Meneruskan seluruh data masukan ke broker MQTT pada topik `Syalannn`.

---

## Panduan Pemasangan Layanan Latar Belakang (Latar Belakang / Service 24/7)

Agar aplikasi dapat berjalan terus-menerus di latar belakang saat komputer menyala tanpa memerlukan jendela terminal aktif, gunakan modul pemasangan di dalam folder `deployment/`.

### 1. Pemasangan di Windows (Startup Application)

> [!WARNING]
> **Penting untuk Windows**: Windows memblokir deteksi keyboard (keystroke hook) dari Service yang berjalan di Session 0. Oleh karena itu, di Windows, aplikasi dipasang sebagai **Startup Application** di sesi user aktif secara tersembunyi (tanpa jendela CMD), bukan sebagai Windows Service sistem biasa.

#### Langkah Pemasangan:
1. Buka folder `c:\Users\Lenovo\Documents\dev\servfor\scanner\deployment` di Command Prompt (CMD) atau File Explorer.
2. Klik ganda atau jalankan file script:
   ```cmd
   install_windows_startup.bat
   ```
3. Script akan mendaftarkan file peluncur otomatis tersembunyi di folder Startup Anda.
4. Setiap kali Anda masuk/login ke Windows, gateway akan langsung berjalan secara tak terlihat di latar belakang.
5. Anda dapat memantau log aktivitasnya secara real-time di file:
   `deployment\gateway_output.log`

---

### 2. Pemasangan di Linux (systemd Service)

Pada sistem operasi Linux, aplikasi dapat didaftarkan sebagai sistem daemon (`systemd` service) secara penuh karena library `keyboard` di Linux membaca input langsung dari perangkat device `/dev/input/`.

#### Langkah Pemasangan:
1. Pindahkan folder `scanner` ke perangkat Linux Anda (misal ke `/opt/usb_scanner`).
2. Masuk ke folder `deployment/` di terminal Linux Anda.
3. Jalankan script instalasi otomatis sebagai root:
   ```bash
   sudo chmod +x install_linux_service.sh
   sudo ./install_linux_service.sh
   ```
4. Perintah di atas akan menginstall dependensi, menyalin file service ke `/etc/systemd/system/`, dan mengaktifkan service 24/7.
5. **Perintah Berguna**:
   - Memantau status service:
     ```bash
     sudo systemctl status usb-scanner.service
     ```
   - Memantau log masukan data secara real-time:
     ```bash
     sudo journalctl -u usb-scanner.service -f
     ```
