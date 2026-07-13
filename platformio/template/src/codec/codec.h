#ifndef CODEC_H
#define CODEC_H

#include <Arduino.h>
#include "frames.h"

class Codec {
public:
    /// Encode TelemetryFrame → binary buffer (16 bytes).
    /// Returns encoded size (0 on error).
    static size_t encodeTelemetry(const TelemetryFrame& frame, uint8_t* out, size_t outLen);

    /// Encode HeartbeatFrame → binary buffer (31 bytes).
    /// Returns encoded size (0 on error).
    static size_t encodeHeartbeat(const HeartbeatFrame& frame, uint8_t* out, size_t outLen);

    /// Decode binary buffer → TelemetryFrame.
    /// Returns true on success.
    static bool decodeTelemetry(const uint8_t* data, size_t len, TelemetryFrame& out);

    /// Decode binary buffer → HeartbeatFrame.
    static bool decodeHeartbeat(const uint8_t* data, size_t len, HeartbeatFrame& out);
};

#endif // CODEC_H
