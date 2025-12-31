Berikut adalah paket lengkap dokumentasi, modul OTA, dan Full Code Final untuk kedua model (M1 & MDCW) yang sudah distandarisasi menggunakan protokol JSON terbaru, Preferences, dan LWT.

1. DOKUMENTASI STANDARISASI (Simpan sebagai IOT_PROTOCOL.md)
Ini adalah "Kitab Suci" komunikasi antara ESP32 dan Dashboard.

Markdown

# IOT STANDARDIZATION PROTOCOL v2.0

## 1. Topik MQTT
Format Topic: `nodes/{NODE_ID}/{FUNCTION}`

| Function  | Topik                  | Arah          | QoS | Keterangan |
| :---      | :---                   | :---          | :-- | :--- |
| **Status**| `nodes/+/status`       | Device -> Cloud | 1 (Retain) | Discovery, Heartbeat, Sync Config. |
| **Monitor**| `nodes/+/monitor`     | Device -> Cloud | 0 | Data Sensor Live, RAM, Error logs. |
| **Command**| `nodes/{ID}/command`  | Cloud -> Device | 1 | Perintah (Config, OTA, Reboot). |
| **Log** | `nodes/{ID}/log`       | Device -> Cloud | 0 | Debugging text string. |

## 2. Struktur JSON

### A. Discovery & Status (`nodes/{ID}/status`)
Payload ini dikirim saat boot, reconnect, atau setelah config berubah.
```json
{
  "id": "CK3-14-1-AABBCC",
  "model": "TEMP_M1",           // "TEMP_M1" atau "MDCW_V1"
  "ver": "2.1.0",
  "state": "online",            // "online" atau "offline" (LWT)
  "ip": "192.168.1.10",
  "conf": {                     // Current Configuration on Device
    "ck": 3,
    "area": 14,
    "no": 1,
    "min": 20.0,
    "max": 25.0,
    "prefix": "cek"             // Khusus MDCW
  }
}
B. Live Monitoring (nodes/{ID}/monitor)
Payload data sensor real-time.

JSON

{
  "id": "CK3-14-1-AABBCC",
  "ts": "2024-01-01 10:00:00",
  "ram": 120000,
  "data": {
    // Jika Model TEMP_M1
    "temp": 24.5,
    "relay": 1
    // Jika Model MDCW_V1
    "total": 100,
    "code": 55,
    "weight": 10
  }
}
C. Control Command (nodes/{ID}/command)
Dashboard mengirim ini ke alat.

JSON

// 1. Ganti Config (Kirim field yang mau diubah saja)
{
  "cmd": "set_config",
  "ck": 5,
  "min": 18.0,
  "prefix": "LINE-A"
}

// 2. OTA Update
{
  "cmd": "ota",
  "url": "[http://domain.com/firmware.bin](http://domain.com/firmware.bin)"
}

// 3. Reboot
{
  "cmd": "reboot"
}
2. MODUL RAW OTA (Reusable Code)
Ini adalah fungsi mentah OTA. Kamu bisa copy-paste fungsi ini ke kode ESP32 manapun.

Syarat Library: #include <HTTPClient.h>, #include <Update.h>

C++

// --- MODUL RAW OTA START ---
void performOTA(const String &url) {
  if (url.length() == 0) return;
  
  Serial.println("[OTA] Starting Update from: " + url);
  
  HTTPClient http;
  // Gunakan WiFiClientSecure jika HTTPS
  WiFiClient client; 
  
  http.begin(client, url);
  int httpCode = http.GET();

  if (httpCode == HTTP_CODE_OK) {
    int contentLength = http.getSize();
    bool canBegin = Update.begin(contentLength);

    if (canBegin) {
      Serial.println("[OTA] Writing firmware...");
      WiFiClient *stream = http.getStreamPtr();
      size_t written = Update.writeStream(*stream);

      if (written == contentLength) {
        Serial.println("[OTA] Written successfully: " + String(written) + "/" + String(contentLength));
      } else {
        Serial.println("[OTA] Written only partial: " + String(written) + "/" + String(contentLength));
      }

      if (Update.end()) {
        if (Update.isFinished()) {
          Serial.println("[OTA] Update Successfully Completed. Rebooting...");
          delay(1000);
          ESP.restart();
        } else {
          Serial.println("[OTA] Update not finished? Something went wrong!");
        }
      } else {
        Serial.println("[OTA] Error Occurred. Error #: " + String(Update.getError()));
      }
    } else {
      Serial.println("[OTA] Not enough space to begin OTA");
    }
  } else {
    Serial.println("[OTA] HTTP Failed, code: " + String(httpCode));
  }
  http.end();
}
// --- MODUL RAW OTA END ---
3. TEMPLATE KODE BARU (Skeleton)
Jika mau buat alat baru, gunakan kerangka ini agar langsung kompatibel dengan dashboard.

C++

#include <Arduino.h>
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <Preferences.h>
// Include module OTA di atas...

// --- SETUP IDENTITAS ---
#define MODEL_NAME "NAMA_ALAT_BARU" // Ganti ini
#define FW_VER     "1.0.0"

// --- SETUP CONFIG ---
Preferences prefs;
// Definisi variabel global config (default value)
int configSatu = 10; 
String configDua = "test";

void setup() {
  // 1. Load Config
  prefs.begin("conf", true);
  configSatu = prefs.getInt("satu", 10);
  configDua = prefs.getString("dua", "test");
  prefs.end();

  // 2. Setup WiFi & MQTT (Standard LWT)
  // ... (Lihat contoh Full Code di bawah)
  
  // 3. Setup Hardware Khusus
  // >>> MASUKKAN SETUP SENSOR DISINI <<<
}

void loop() {
  // 1. Maintain Connection
  // ...
  
  // 2. Logic Utama
  // >>> MASUKKAN LOGIC BACA SENSOR DISINI <<<
  
  // 3. Publish Data
  // Gunakan format JSON "monitor"
}

void mqttCallback(...) {
  // Handle "set_config" -> Update variabel -> Save Preferences
}