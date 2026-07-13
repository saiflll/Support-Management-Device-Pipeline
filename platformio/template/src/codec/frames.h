#ifndef FRAMES_H
#define FRAMES_H

#include <Arduino.h>
#include <stdint.h>

// ── Constants ──────────────────────────────────────────────────
#define TELEMETRY_VERSION   1
#define TELEMETRY_MSGTYPE   0x01
#define HEARTBEAT_VERSION   1
#define HEARTBEAT_MSGTYPE   0x02

#define MAX_OFFLINE_BUFFER  500

// ── TelemetryFrame (18 bytes payload + 2 CRC = 20) ─────────────
#pragma pack(push, 1)
typedef struct {
    uint8_t  version;    // 0: =1
    uint8_t  msgType;    // 1: =0x01
    uint16_t seq;        // 2-3: sequence
    uint32_t ts;         // 4-7: unix timestamp
    int16_t  ck;         // 8-9: central kitchen ID
    int16_t  area;       // 10-11: area ID
    int16_t  total;      // 12-13: total produk (reg2)
    int16_t  code;       // 14-15: kode MD (reg5)
    int16_t  weight;     // 16-17: berat (reg114)
    uint16_t crc16;      // 18-19: CRC16 Modbus
} TelemetryFrame;

// ── HeartbeatFrame (33 bytes + 2 CRC = 35) ────────────────────
typedef struct {
    uint8_t  version;    // 0: =1
    uint8_t  msgType;    // 1: =0x02
    uint16_t seq;        // 2-3: sequence
    uint32_t ts;         // 4-7: unix timestamp
    uint32_t uptime;     // 8-11: seconds since boot
    uint32_t heap;       // 12-15: free heap
    uint32_t ipAddr;     // 16-19: IP packed
    int8_t   rssi;       // 20: WiFi RSSI dBm
    int16_t  cpuTempX10; // 21-22: core temp ×10 (425 = 42.5°C)
    uint8_t  resetReason;// 23: 0=poweron,1=wdt,2=panic,3=sw,4=ota,5=other
    uint16_t interval;   // 24-25: device publish interval (ms)
    uint8_t  fwMajor;    // 26
    uint8_t  fwMinor;    // 27
    int16_t  ck;         // 28-29: central kitchen ID
    int16_t  area;       // 30-31: area ID
    uint16_t crc16;      // 32-33: CRC16 Modbus
} HeartbeatFrame;

// ── Offline buffer entry ───────────────────────────────────────
typedef struct {
    uint32_t ts;
    int16_t  total;
    int16_t  code;
    int16_t  weight;
} TelemetryEntry;

#pragma pack(pop)

// ── CRC16 Modbus ───────────────────────────────────────────────
class CRC16 {
public:
    static uint16_t compute(const uint8_t* data, size_t len);
};

#endif // FRAMES_H
