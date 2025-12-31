#include <Arduino.h>
#include <WiFi.h>
#include <PubSubClient.h>
#include <HTTPClient.h>
#include <Update.h>
#include <ArduinoJson.h>
#include <Preferences.h>

// ==========================================
// 1. EDIT BAGIAN INI SESUAI KEBUTUHAN
// ==========================================
#define MODEL_NAME      "GENERIC_SENSOR_V1" // Nama Model Alat
#define FW_VERSION      "1.0.0"             // Versi Firmware
#define DEFAULT_INT     5000                // Default Interval (ms)

const char* WIFI_SSID   = "NAMA_WIFI";
const char* WIFI_PASS   = "PASS_WIFI";
const char* MQTT_SERVER = "192.168.1.100";  // IP Broker
const int   MQTT_PORT   = 1883;
// Jika ada user/pass mqtt:
const char* MQTT_USER   = ""; 
const char* MQTT_PASS   = ""; 

// ==========================================
// 2. VARIABEL GLOBAL SYSTEM (JANGAN UBAH)
// ==========================================
WiFiClient espClient;
PubSubClient mqtt(espClient);
Preferences prefs;
String nodeID;
unsigned long lastLoop = 0;

// Variabel Config (Bisa diubah via Dashboard)
struct Config {
  unsigned long interval;
  float threshold; // Contoh variabel settingan
  String note;     // Contoh variabel text
} sysConfig;

// ==========================================
// 3. MODUL OTA (SUDAH LENGKAP)
// ==========================================
void performOTA(const String &url) {
  if (url.length() == 0) return;
  Serial.println("[OTA] Update start: " + url);
  
  WiFiClient client;
  HTTPClient http;
  http.begin(client, url);
  
  int httpCode = http.GET();
  if (httpCode == 200) {
    int contentLength = http.getSize();
    bool canBegin = Update.begin(contentLength);
    if (canBegin) {
      Serial.println("[OTA] Downloading & Writing...");
      size_t written = Update.writeStream(*http.getStreamPtr());
      if (written == contentLength && Update.end(true)) {
        Serial.println("[OTA] Success! Rebooting...");
        delay(1000);
        ESP.restart();
      } else {
        Serial.println("[OTA] Error: " + String(Update.getError()));
      }
    } else {
      Serial.println("[OTA] Not enough space");
    }
  } else {
    Serial.println("[OTA] HTTP Error: " + String(httpCode));
  }
  http.end();
}

// ==========================================
// 4. CONFIG MANAGER (SUDAH LENGKAP)
// ==========================================
void loadConfig() {
  prefs.begin("sys_conf", true); // Read Only
  sysConfig.interval = prefs.getULong("int", DEFAULT_INT);
  sysConfig.threshold = prefs.getFloat("thr", 0.0);
  sysConfig.note = prefs.getString("note", "default");
  prefs.end();
  Serial.printf("[CFG] Loaded: Int=%lu, Thr=%.2f\n", sysConfig.interval, sysConfig.threshold);
}

void saveConfig() {
  prefs.begin("sys_conf", false); // Read Write
  prefs.putULong("int", sysConfig.interval);
  prefs.putFloat("thr", sysConfig.threshold);
  prefs.putString("note", sysConfig.note);
  prefs.end();
  Serial.println("[CFG] Saved to Flash");
}

// ==========================================
// 5. MQTT HELPER (SUDAH LENGKAP)
// ==========================================
void publishStatus() {
  JsonDocument doc;
  doc["id"] = nodeID;
  doc["model"] = MODEL_NAME;
  doc["ver"] = FW_VERSION;
  doc["state"] = "online";
  doc["ip"] = WiFi.localIP().toString();
  
  // Sync Config ke Dashboard
  JsonObject conf = doc["conf"].to<JsonObject>();
  conf["interval"] = sysConfig.interval;
  conf["threshold"] = sysConfig.threshold;
  conf["note"] = sysConfig.note;

  char buf[512];
  serializeJson(doc, buf);
  // Retain = True agar dashboard baru buka langsung tau status
  mqtt.publish(("nodes/" + nodeID + "/status").c_str(), buf, true);
}

void mqttCallback(char* topic, byte* payload, unsigned int length) {
  JsonDocument doc;
  DeserializationError err = deserializeJson(doc, payload);
  if (err) return;

  const char* cmd = doc["cmd"];
  if (!cmd) return;

  Serial.println("[CMD] Recv: " + String(cmd));

  // --- CMD: SET CONFIG ---
  if (strcmp(cmd, "set_config") == 0) {
    bool changed = false;
    if (doc.containsKey("interval")) { sysConfig.interval = doc["interval"]; changed = true; }
    if (doc.containsKey("threshold")) { sysConfig.threshold = doc["threshold"]; changed = true; }
    if (doc.containsKey("note")) { sysConfig.note = doc["note"].as<String>(); changed = true; }
    
    if (changed) {
      saveConfig();
      publishStatus(); // Lapor config baru
    }
  
  // --- CMD: OTA ---
  } else if (strcmp(cmd, "ota") == 0) {
    performOTA(doc["url"].as<String>());
  
  // --- CMD: REBOOT ---
  } else if (strcmp(cmd, "reboot") == 0) {
    ESP.restart();
  }
}

void mqttReconnect() {
  while (!mqtt.connected()) {
    Serial.print("Connecting MQTT...");
    String clientId = String(MODEL_NAME) + "-" + nodeID;
    
    // LWT (Last Will) - Pesan Kematian
    String lwtTopic = "nodes/" + nodeID + "/status";
    String lwtMsg = "{\"id\":\"" + nodeID + "\",\"state\":\"offline\",\"model\":\"" + String(MODEL_NAME) + "\"}";

    if (mqtt.connect(clientId.c_str(), MQTT_USER, MQTT_PASS, lwtTopic.c_str(), 1, true, lwtMsg.c_str())) {
      Serial.println(" OK!");
      mqtt.subscribe(("nodes/" + nodeID + "/command").c_str());
      publishStatus(); // Say hello
    } else {
      Serial.print(" Failed rc="); Serial.print(mqtt.state());
      delay(3000);
    }
  }
}

// ==========================================
// 6. MAIN SETUP
// ==========================================
void setup() {
  Serial.begin(115200);
  delay(500);

  // 1. Config & ID
  loadConfig();
  String mac = WiFi.macAddress();
  mac.replace(":", "");
  nodeID = "NODE-" + mac; // ID Unik Otomatis

  // 2. WiFi
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  Serial.print("WiFi Connecting");
  while (WiFi.status() != WL_CONNECTED) {
    delay(500); Serial.print(".");
  }
  Serial.println("\nIP: " + WiFi.localIP().toString());

  // 3. MQTT
  mqtt.setServer(MQTT_SERVER, MQTT_PORT);
  mqtt.setCallback(mqttCallback);

  // >>> MASUKKAN SETUP SENSOR DI SINI <<<
  // pinMode(PIN_SENSOR, INPUT);
}

// ==========================================
// 7. MAIN LOOP
// ==========================================
void loop() {
  if (!mqtt.connected()) mqttReconnect();
  mqtt.loop();

  unsigned long now = millis();
  if (now - lastLoop >= sysConfig.interval) {
    lastLoop = now;

    // >>> 1. BACA SENSOR DI SINI <<<
    float val1 = random(20, 30); // Contoh Dummy
    int val2 = 1;                // Contoh Dummy

    // >>> 2. KIRIM DATA (MONITOR) <<<
    JsonDocument doc;
    doc["id"] = nodeID;
    doc["ts"] = now;
    doc["ram"] = ESP.getFreeHeap();
    
    // Data Payload
    JsonObject data = doc["data"].to<JsonObject>();
    data["sensor_1"] = val1;
    data["sensor_2"] = val2;
    data["config_thr"] = sysConfig.threshold; // Feedback nilai threshold

    char buf[512];
    serializeJson(doc, buf);
    mqtt.publish(("nodes/" + nodeID + "/monitor").c_str(), buf);
    
    Serial.println("[MONITOR] Sent: " + String(buf));
  }
}