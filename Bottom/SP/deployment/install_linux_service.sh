#!/bin/bash

# Pastikan script dijalankan sebagai root
if [ "$EUID" -ne 0 ]; then
  echo "[-] Harap jalankan script ini sebagai root (sudo)."
  exit 1
fi

echo "[+] Memulai instalasi USB Scanner Service..."

# 1. Buat direktori kerja
mkdir -p /opt/usb_scanner

# 2. Salin file script Python dan dependensinya
# Meniru letak file dari folder pengembangan saat dipindahkan ke Linux
cp ../usb_scanner.py /opt/usb_scanner/

# 3. Install requirements
echo "[+] Menginstall dependensi python (keyboard, pyserial, paho-mqtt)..."
pip3 install keyboard pyserial paho-mqtt

# 4. Salin file service systemd
cp usb-scanner.service /etc/systemd/system/

# 5. Reload daemon systemd
echo "[+] Mendaftarkan service ke systemd..."
systemctl daemon-reload

# 6. Enable service agar jalan otomatis saat boot
systemctl enable usb-scanner.service

# 7. Jalankan service sekarang
echo "[+] Menjalankan service..."
systemctl start usb-scanner.service

echo "[✓] Instalasi selesai!"
echo "[i] Gunakan perintah ini untuk melihat status service:"
echo "    sudo systemctl status usb-scanner.service"
echo "[i] Gunakan perintah ini untuk memantau log secara real-time:"
echo "    sudo journalctl -u usb-scanner.service -f"
