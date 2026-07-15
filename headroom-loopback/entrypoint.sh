#!/bin/sh

# Start nginx in background
nginx

# Run headroom proxy (blocking)
headroom proxy --port 8787 --host 0.0.0.0
