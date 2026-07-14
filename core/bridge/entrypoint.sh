#!/bin/sh

# Jalankan headroom proxy di background
# --host 0.0.0.0 agar bisa diakses dari luar container (oleh service forwarder)
headroom proxy --port 8787 --host 0.0.0.0 &

# Jalankan aplikasi bridge utama
/bridge
