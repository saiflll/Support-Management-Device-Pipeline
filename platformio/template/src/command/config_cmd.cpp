// ── Config Command Handler ─────────────────────────────────────
// Command MQTT:
//   {"cmd":"set_config", "config":{"interval":3000, "prefix":"MDCW1 (UK)"}}
//   {"cmd":"get_config"}
//   {"cmd":"reboot"}

#include <ArduinoJson.h>
#include "../config/config.h"
#include "../monitor/heartbeat.h"
#include "../topics/topics.h"
#include "../system/logger.h"

void handleSetConfig(JsonDocument& doc) {
    JsonObject cfg = doc["config"];
    if (cfg.isNull()) {
        Logger::warn("CMD: set_config missing 'config' object");
        return;
    }

    if (cfg.containsKey("interval"))
        Config::cfg.interval = cfg["interval"] | Config::cfg.interval;
    if (cfg.containsKey("area"))
        Config::cfg.area = cfg["area"] | Config::cfg.area;
    if (cfg.containsKey("prefix")) {
        Config::cfg.prefix = cfg["prefix"].as<String>();
        // Reconfigure topics with new prefix
        Topics::configure(Config::cfg.area, Config::cfg.prefix);
    }

    Config::save();
    Logger::info("CMD: config updated");

    // Publish heartbeat immediately to reflect changes
    Heartbeat::publishNow();
}

void handleGetConfig() {
    // Publish current config as heartbeat (includes version, interval, etc.)
    Heartbeat::publishNow();
}

void handleReboot() {
    Logger::info("CMD: rebooting...");
    delay(100);
    ESP.restart();
}
