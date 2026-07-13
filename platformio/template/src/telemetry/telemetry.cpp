#include "telemetry.h"
#include "../codec/codec.h"
#include "../network/mqtt.h"
#include "../topics/topics.h"
#include "../config/config.h"
#include "../system/logger.h"

// ── Ring Buffer ────────────────────────────────────────────────
TelemetryEntry Telemetry::s_buffer[MAX_OFFLINE_BUFFER];
uint16_t Telemetry::s_head = 0;
uint16_t Telemetry::s_tail = 0;
uint16_t Telemetry::s_count = 0;
uint16_t Telemetry::s_seq = 0;

void Telemetry::init() {
    s_head = 0;
    s_tail = 0;
    s_count = 0;
}

void Telemetry::publish(int16_t total, int16_t code, int16_t weight, uint32_t ts) {
    s_seq++;  // increment global sequence

    if (!MqttNet::isConnected()) {
        // Offline → buffer to ring
        if (s_count < MAX_OFFLINE_BUFFER) {
            s_buffer[s_tail] = {ts, total, code, weight};
            s_tail = (s_tail + 1) % MAX_OFFLINE_BUFFER;
            s_count++;
        } else {
            Logger::warn("Offline buffer FULL, dropping entry");
        }
        return;
    }

    // ── Binary telemetry frame (decoded by bridge → backend) ──
    TelemetryFrame f;
    f.version = TELEMETRY_VERSION;
    f.msgType = TELEMETRY_MSGTYPE;
    f.seq     = s_seq;
    f.ts      = ts;
    f.ck      = Config::cfg.area;     // uses area as default CK
    f.area    = Config::cfg.area;
    f.total   = total;
    f.code    = code;
    f.weight  = weight;
    f.crc16   = 0;

    uint8_t out[20];
    size_t len = Codec::encodeTelemetry(f, out, sizeof(out));
    if (len > 0) {
        MqttNet::publish(Topics::telemetry().c_str(), out, len, false);
    }
}

void Telemetry::flushOffline() {
    if (s_count == 0 || !MqttNet::isConnected()) return;

    uint16_t flushed = 0;
    while (s_count > 0 && MqttNet::isConnected()) {
        TelemetryEntry& e = s_buffer[s_head];

        // Replay — binary telemetry frame
        Telemetry::publish(e.total, e.code, e.weight, e.ts);

        s_head = (s_head + 1) % MAX_OFFLINE_BUFFER;
        s_count--;
        flushed++;
        delay(5);
    }

    if (flushed > 0) {
        Logger::info("Flushed %u offline telemetry entries", flushed);
    }
}
