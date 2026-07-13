#include "heartbeat.h"
#include "../telemetry/telemetry.h"
#include "../codec/codec.h"
#include "../codec/frames.h"
#include "../network/mqtt.h"
#include "../topics/topics.h"
#include "../config/config.h"
#include "../diagnostic/diagnostic.h"

unsigned long Heartbeat::s_lastBeat = 0;

void Heartbeat::init() {
    s_lastBeat = millis();
}

void Heartbeat::publishTick(unsigned long nowMs) {
    if (nowMs - s_lastBeat < BEAT_INTERVAL) return;
    s_lastBeat = nowMs;

    if (MqttNet::isConnected()) {
        doPublish();
    }
}

void Heartbeat::publishNow() {
    if (MqttNet::isConnected()) {
        doPublish();
    }
}

void Heartbeat::doPublish() {
    // ── Binary heartbeat frame (retained, decoded by bridge → OTA topics) ──
    HeartbeatFrame f;
    f.version     = HEARTBEAT_VERSION;
    f.msgType     = HEARTBEAT_MSGTYPE;
    f.seq         = Telemetry::getSeq();
    f.ts          = (uint32_t)time(nullptr);
    f.uptime      = millis() / 1000;
    f.heap        = Diagnostic::freeHeap();
    f.ipAddr      = (uint32_t)WiFi.localIP();
    f.rssi        = (int8_t)WiFi.RSSI();
    f.cpuTempX10  = Diagnostic::cpuTempX10();
    f.resetReason = Diagnostic::resetReasonEnum();
    f.interval    = Config::cfg.interval;
    f.fwMajor     = FW_MAJOR;
    f.fwMinor     = FW_MINOR;
    f.ck          = Config::cfg.area;
    f.area        = Config::cfg.area;
    f.crc16       = 0;

    uint8_t out[35];
    size_t len = Codec::encodeHeartbeat(f, out, sizeof(out));
    if (len > 0) {
        MqttNet::publish(Topics::heartbeat().c_str(), out, len, true);  // retained
    }
}
