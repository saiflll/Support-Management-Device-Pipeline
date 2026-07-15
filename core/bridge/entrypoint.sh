#!/bin/sh

# Headroom proxy (berjalan tanpa auth secara default)
# Dashboard: http://localhost:8787/dashboard
# Readiness: http://localhost:8787/readyz
headroom proxy --port 8787 --host 0.0.0.0 &

# Tunggu sebentar memastikan headroom siap
sleep 3

# Jalankan aplikasi bridge utama
/bridge
