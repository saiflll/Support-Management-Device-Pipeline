#!/bin/sh

# Jalankan headroom proxy di background
# --no-auth digunakan agar kita tidak perlu API Key untuk penggunaan lokal/internal
headroom proxy --port 8787 --host 0.0.0.0 --no-auth &

# Jalankan aplikasi bridge utama
/bridge
