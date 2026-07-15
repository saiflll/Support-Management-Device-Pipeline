#!/bin/sh

# Rust _core.so gak kompatibel sama KVM lawas → pake Python-only mode.
export HEADROOM_REQUIRE_RUST_CORE=false

# Start nginx in background
nginx

# Run headroom proxy (blocking)
headroom proxy --port 8787 --host 0.0.0.0
