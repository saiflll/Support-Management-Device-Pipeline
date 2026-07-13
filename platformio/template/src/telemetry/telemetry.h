#ifndef TELEMETRY_H
#define TELEMETRY_H

#include <Arduino.h>
#include "../codec/frames.h"

class Telemetry {
public:
    static void init();

    /// Publish telemetry data.
    /// If MQTT offline → buffer to ring. If online → encode binary & publish immediately.
    static void publish(int16_t total, int16_t code, int16_t weight, uint32_t ts);

    /// Flush offline buffer to MQTT (call in loop when connected).
    static void flushOffline();

    /// Get shared sequence number.
    static uint16_t getSeq() { return s_seq; }

    /// Check if buffer has queued entries.
    static bool hasOfflineData() { return s_count > 0; }
    static uint16_t offlineCount() { return s_count; }

private:
    static TelemetryEntry s_buffer[MAX_OFFLINE_BUFFER];
    static uint16_t s_head;     // read index
    static uint16_t s_tail;     // write index
    static uint16_t s_count;    // entries in buffer
    static uint16_t s_seq;      // shared sequence counter
};

#endif // TELEMETRY_H
