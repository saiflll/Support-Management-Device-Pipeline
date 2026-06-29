#!/bin/sh
# Entrypoint: Jalankan kedua binary secara paralel
# Jika salah satu mati, container ini tetap hidup sampai keduanya mati

echo "🚀 Starting IoT Support services..."

# Jalankan forwarder di background
/app/forwarder-server &
FORWARDER_PID=$!
echo "✅ Forwarder started (PID: $FORWARDER_PID)"

# Jalankan ota-server di background
/app/ota-server &
OTA_PID=$!
echo "✅ OTA Server started (PID: $OTA_PID)"

# Tunggu salah satu process mati, lalu hentikan yang lain
wait -n $FORWARDER_PID $OTA_PID
EXIT_CODE=$?

echo "⚠️  One of the services exited. Stopping all..."
kill $FORWARDER_PID $OTA_PID 2>/dev/null
wait

exit $EXIT_CODE
