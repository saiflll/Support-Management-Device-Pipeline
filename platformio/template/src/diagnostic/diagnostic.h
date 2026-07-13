#ifndef DIAGNOSTIC_H
#define DIAGNOSTIC_H

#include <Arduino.h>

/// Reset reason codes (for HeartbeatFrame.resetReason)
enum ResetReason : uint8_t {
    RESET_POWERON  = 0,
    RESET_WDT      = 1,
    RESET_PANIC    = 2,
    RESET_SW       = 3,
    RESET_OTA      = 4,
    RESET_OTHER    = 5
};

class Diagnostic {
public:
    /// Get the ESP32 reset reason and map to our enum.
    static ResetReason resetReasonEnum() {
        esp_reset_reason_t reason = esp_reset_reason();
        switch (reason) {
            case ESP_RST_POWERON:      return RESET_POWERON;
            case ESP_RST_TASK_WDT:
            case ESP_RST_WDT:          return RESET_WDT;
            case ESP_RST_PANIC:        return RESET_PANIC;
            case ESP_RST_SW:           return RESET_SW;
            case ESP_RST_SW_CPU_RESET:
            case ESP_RST_EXT_CPU_RESET: return RESET_OTHER;
            default:                   return RESET_OTHER;
        }
    }

    /// Get the human-readable reset reason string.
    static const char* resetReasonStr() {
        switch (resetReasonEnum()) {
            case RESET_POWERON: return "POWERON";
            case RESET_WDT:     return "WDT";
            case RESET_PANIC:   return "PANIC";
            case RESET_SW:      return "SW_RESET";
            case RESET_OTA:     return "OTA";
            default:            return "OTHER";
        }
    }

    /// Get free heap in bytes.
    static uint32_t freeHeap() {
        return ESP.getFreeHeap();
    }

    /// Get CPU core temperature (×10).
    static int16_t cpuTempX10() {
        return (int16_t)(temperatureRead() * 10.0f);
    }
};

#endif // DIAGNOSTIC_H
