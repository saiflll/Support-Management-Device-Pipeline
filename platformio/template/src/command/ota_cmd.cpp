// ── OTA Command Handler ────────────────────────────────────────
// Dipanggil dari application.cpp saat MQTT message diterima.
//
// Command MQTT:
//   {"cmd":"ota", "url":"http://server/fw.bin", "sha256":"a1b2c3...64hex..."}

#include <ArduinoJson.h>
#include "../ota/ota.h"
#include "../system/logger.h"

void handleOTA(JsonDocument& doc) {
    if (!doc["url"].is<const char*>()) {
        Logger::warn("OTA cmd missing 'url'");
        return;
    }

    String url = doc["url"].as<String>();
    String sha256 = doc["sha256"].as<String>();  // optional

    Logger::info("CMD: OTA url=%s", url.c_str());
    OTA::perform(url, sha256);
}
