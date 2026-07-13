// ============================================================
//  ServFor MDCW — Application Entry Point
//  Board:  ESP32 Dev Module
//  Stack:  Arduino framework + PlatformIO
//  Binary: MQTT telemetry (14B) + heartbeat (29B) + OTA
//  Broker: EMQX 5.5
// ============================================================

#include <Arduino.h>
#include <WiFi.h>
#include <time.h>
#include <ArduinoJson.h>
#include <esp_ota_ops.h>

#include "config/config.h"
#include "network/mqtt.h"
#include "topics/topics.h"
#include "telemetry/telemetry.h"
#include "monitor/heartbeat.h"
#include "ota/ota.h"
#include "system/watchdog.h"
#include "system/logger.h"
#include "diagnostic/diagnostic.h"
#include "command/ota_cmd.h"
#include "command/config_cmd.h"

// ── Forward Declarations ──────────────────────────────────────
static void connectWiFi();
static void onMqttMessage(const String& topic, const uint8_t* payload, size_t len);
static void publishDiagnosticInfo();

// ── State ─────────────────────────────────────────────────────
static bool s_otaRollbackDone = false;

// ══════════════════════════════════════════════════════════════
//  SETUP
// ══════════════════════════════════════════════════════════════
void setup() {
#ifdef SERIAL_DEBUG
    Serial.begin(115200);
    delay(1000);
    Serial.println("\n\n=== ServFor MDCW 1.0 ===");
#endif

    // ── Watchdog (15s) ──
    Watchdog::init();
    Logger::info("Watchdog initialized (%ds)", WDT_TIMEOUT_SECONDS);

    // ── NVS Config ──
    Config::init();
    Logger::info("Config loaded: area=%d, interval=%dms, prefix=%s",
                 Config::cfg.area, Config::cfg.interval, Config::cfg.prefix.c_str());

    // ── Node identity ──
    String nodeId = Config::cfg.prefix;
    nodeId.replace(" ", "_");
    nodeId += "-";
    nodeId += String((uint32_t)(ESP.getEfuseMac() & 0xFFFFFFFF), HEX);

    // ── Topics ──
    Topics::configure(nodeId);
    Logger::info("Telemetry topic: %s", Topics::telemetry().c_str());
    Logger::info("Heartbeat topic: %s", Topics::heartbeat().c_str());
    Logger::info("Command topic: %s", Topics::command().c_str());

    // ── Telemetry ring buffer ──
    Telemetry::init();

    // ── Heartbeat ──
    Heartbeat::init();

    // ── WiFi ──
    connectWiFi();

    // ── MQTT ──
    MqttNet::begin(
        Config::cfg.mqttBroker.c_str(),
        Config::cfg.mqttPort,
        Config::cfg.mqttUser.c_str(),
        Config::cfg.mqttPass.c_str(),
        nodeId.c_str()
    );
    MqttNet::onMessage(onMqttMessage);
    MqttNet::reconnect();
    MqttNet::subscribe(Topics::command().c_str());

    // ── OTA Rollback Safety ──
    const esp_partition_t* running = esp_ota_get_running_partition();
    esp_ota_img_states_t otaState;
    if (esp_ota_get_state_partition(running, &otaState) == ESP_OK) {
        if (otaState == ESP_OTA_IMG_PENDING_VERIFY) {
            Logger::info("OTA: firmware in PENDING_VERIFY — will validate on MQTT connect");
        }
    }

    // ── NTP ──
    configTime(7 * 3600, 0, "pool.ntp.org", "time.google.com");

    Logger::info("=== Setup complete ===");
}

// ══════════════════════════════════════════════════════════════
//  LOOP
// ══════════════════════════════════════════════════════════════
void loop() {
    Watchdog::feed();

    // WiFi
    if (WiFi.status() != WL_CONNECTED) {
        connectWiFi();
    }

    // MQTT
    MqttNet::loop();

    // Heartbeat (30s)
    Heartbeat::publishTick(millis());

    // Flush offline telemetry buffer
    Telemetry::flushOffline();

    // ── OTA Rollback: validate when MQTT first connects after boot ──
    if (!s_otaRollbackDone && MqttNet::isConnected()) {
        s_otaRollbackDone = true;
        const esp_partition_t* running = esp_ota_get_running_partition();
        esp_ota_img_states_t state;
        if (esp_ota_get_state_partition(running, &state) == ESP_OK) {
            if (state == ESP_OTA_IMG_PENDING_VERIFY) {
                esp_ota_mark_app_valid_cancel_rollback();
                Logger::info("OTA: firmware validated (MQTT OK), rollback cancelled");
                // Publish heartbeat immediately to reflect validated state
                Heartbeat::publishNow();
            }
        }
    }

    // ── TODO: Insert MDCW modbus read + telemetry publish here ──
    // Example (from real hardware):
    //   ModbusDriver::loop();
    //   static unsigned long lastTel = 0;
    //   if (millis() - lastTel >= Config::cfg.interval) {
    //       lastTel = millis();
    //       int16_t total = Modbus.getReg2();
    //       int16_t code  = Modbus.getReg5();
    //       int16_t weight = Modbus.getReg114();
    //       Telemetry::publish(total, code, weight, (uint32_t)time(nullptr));
    //   }
}

// ══════════════════════════════════════════════════════════════
//  WiFi
// ══════════════════════════════════════════════════════════════
static void connectWiFi() {
    Logger::info("WiFi: connecting to %s...", Config::cfg.ssid.c_str());

    WiFi.mode(WIFI_STA);
    WiFi.begin(Config::cfg.ssid.c_str(), Config::cfg.pass.c_str());

    unsigned long start = millis();
    while (WiFi.status() != WL_CONNECTED && (millis() - start) < 20000) {
        delay(500);
        Watchdog::feed();
        Serial.print(".");
    }

    if (WiFi.status() == WL_CONNECTED) {
        Logger::info("WiFi: connected, IP=%s", WiFi.localIP().toString().c_str());
        configTime(7 * 3600, 0, "pool.ntp.org", "time.google.com");
    } else {
        Logger::error("WiFi: FAILED");
    }
}

// ══════════════════════════════════════════════════════════════
//  MQTT Callback
// ══════════════════════════════════════════════════════════════
static void onMqttMessage(const String& topic, const uint8_t* payload, size_t len) {
    // Convert to null-terminated string for JSON parsing
    String msg;
    msg.reserve(len + 1);
    for (size_t i = 0; i < len; i++) msg += (char)payload[i];

    Logger::info("MQTT << %s: %s", topic.c_str(), msg.c_str());

    // ── OTA + Config command ──
    if (topic == Topics::command()) {
        JsonDocument doc;
        DeserializationError err = deserializeJson(doc, msg);
        if (!err) {
            const char* cmd = doc["cmd"];
            if (!cmd) return;
            if (strcmp(cmd, "ota") == 0) {
                handleOTA(doc);
            } else if (strcmp(cmd, "set_config") == 0) {
                handleSetConfig(doc);
            } else if (strcmp(cmd, "get_config") == 0) {
                handleGetConfig();
            } else if (strcmp(cmd, "reboot") == 0) {
                handleReboot();
            }
        }
    }

    // ── System connected notification (internal) ──
    if (topic == "system/connected") {
        // MQTT just connected — resubscribe command topic + flush offline
        MqttNet::subscribe(Topics::command().c_str());
        Heartbeat::publishNow();
        Telemetry::flushOffline();
    }
}
