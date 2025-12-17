#!/bin/bash

# Clear screen for fresh start
clear

echo "========================================================"
echo "  IOT SERVICE LAUNCHER (LINUX PROD - SAFE MODE)"
echo "========================================================"
echo

# 1. Build & Up (Smart Update)
# Note: We do NOT use 'docker-compose down' here to preserve running containers
# that haven't changed. 'up -d --build' will only recreate changed containers.
echo "[1/2] Updating Services (Smart Build)..."
echo "      (Only changed services will be recreated)"
echo

export BUILDKIT_PROGRESS=tty
if ! docker-compose up -d --build --remove-orphans; then
    echo
    echo "      [ERROR] Failed to start services."
    exit 1
fi

# 2. Wait & Check
echo
echo "[2/2] Verifying (5s)..."
sleep 5
clear

echo "========================================================"
echo "  PRODUCTION STATUS"
echo "========================================================"
docker-compose ps --format "table {{.Name}}\t{{.State}}\t{{.Status}}\t{{.Ports}}"

echo
echo "========================================================"
echo "View logs: docker-compose logs -f [service_name]"
echo