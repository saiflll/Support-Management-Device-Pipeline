#!/bin/bash
# ============================================================
# EMQX Bridge Initializer
# Memanggil EMQX REST API untuk membuat data bridge ke cloud
# ============================================================
# Dipanggil oleh: deploy.yml (GitHub Actions) atau manual setelah EMQX start
# Usage: ./init-bridges.sh [emqx_host] [emqx_api_key] [emqx_api_secret]
# ============================================================

set -euo pipefail

EMQX_HOST="${1:-http://emqx:18083}"
EMQX_API_KEY="${2:-}"
EMQX_API_SECRET="${3:-}"

# Baca dari .env jika tidak ada argumen
if [ -z "$EMQX_API_KEY" ] && [ -f .env ]; then
    source .env 2>/dev/null || true
fi

# Cek dependencies
if ! command -v curl &>/dev/null; then
    echo "❌ curl required. Install with: apk add curl"
    exit 1
fi
if ! command -v jq &>/dev/null; then
    echo "❌ jq required. Install with: apk add jq"
    exit 1
fi

echo "=========================================="
echo " EMQX Bridge Initializer"
echo "=========================================="
echo "Host: $EMQX_HOST"
echo ""

# Tunggu EMQX siap
echo "⏳ Waiting for EMQX to be ready..."
for i in $(seq 1 30); do
    if curl -sf "$EMQX_HOST/api/v5/status" > /dev/null 2>&1; then
        echo "✅ EMQX is ready!"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "❌ EMQX did not become ready in time"
        exit 1
    fi
    sleep 2
done

# Buat dashboard user untuk API key (jika perlu)
# Sebenarnya EMQX default sudah punya admin:public, tapi lebih baik pake API Key

AUTH_HEADER=""
if [ -n "$EMQX_API_KEY" ] && [ -n "$EMQX_API_SECRET" ]; then
    AUTH_HEADER="-u ${EMQX_API_KEY}:${EMQX_API_SECRET}"
fi

# ============================================
# 1. Buat MQTT Data Bridge ke Cloud
# ============================================
# Tujuan: Forward data dari topic forwarder/outgoing ke cloud MQTT

BRIDGE_NAME="mqtt_cloud_forward"
BRIDGE_CONFIG=$(cat <<EOF
{
  "name": "${BRIDGE_NAME}",
  "type": "mqtt",
  "enable": true,
  "server": "${MQTT_BROKER_PUB:-tcp://emqx.miegacoan.id:1883}",
  "username": "${MQTT_USERNAME_FOR:-forward}",
  "password": "${MQTT_PASSWORD_FOR:-forward}",
  "direction": "egress",
  "remote_topic": "${MQTT_TOPIC_PUB:-sensor/data/ingest}",
  "local_topic": "forwarder/outgoing",
  "retain": false,
  "qos": 1,
  "resource_opts": {
    "start_timeout_ms": 5000,
    "health_check_interval_ms": 30000,
    "query_mode": "sync",
    "inflight_window": 100,
    "worker_pool_size": 4,
    "health_check_method": "mqtt",
    "auto_restart_interval_ms": 5000
  }
}
EOF
)

echo "📤 Creating/updating MQTT Data Bridge: $BRIDGE_NAME..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "${EMQX_HOST}/api/v5/bridges/mqtt%3A${BRIDGE_NAME}" \
    -H "Content-Type: application/json" \
    ${AUTH_HEADER} \
    -d "$BRIDGE_CONFIG" 2>&1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo "✅ Data Bridge '$BRIDGE_NAME' berhasil dibuat/diperbarui!"
elif [ "$HTTP_CODE" = "409" ]; then
    echo "⚠️  Data Bridge '$BRIDGE_NAME' sudah ada (409 Conflict) — mungkin perlu diupdate manual"
else
    echo "❌ Gagal membuat Data Bridge. HTTP code: $HTTP_CODE"
    echo "   Coba buat manual di EMQX Dashboard → Integrations → Bridges"
fi

echo ""
echo "=========================================="
echo " ✅ EMQX Bridge initialization complete!"
echo "=========================================="
echo ""
echo "Topic mapping:"
echo "  Local  forwarder/outgoing → MQTT Bridge → Cloud ${MQTT_TOPIC_PUB:-sensor/data/ingest}"
echo ""
echo "To verify: curl $EMQX_HOST/api/v5/bridges/mqtt%3A${BRIDGE_NAME}/metrics"
