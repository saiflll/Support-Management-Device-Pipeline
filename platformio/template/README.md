# ServFor — PlatformIO Bare-Metal Template

ESP32 firmware: binary telemetry (14B) + heartbeat (29B) + OTA dengan SHA256 + rollback safety.

## Struktur

```
src/
├── application.cpp         # Entry point (setup/loop)
├── telemetry/              # publish() + ring buffer 500 entry + flush offline
├── monitor/heartbeat.cpp   # 30s periodik, retained, system stats
├── ota/ota.cpp             # HTTP → flash, SHA256 verify, ESP rollback
├── command/
│   ├── ota_cmd.cpp         # Handler {"cmd":"ota","url":"..."}
│   └── config_cmd.cpp      # Handler set_config/get_config/reboot
├── network/mqtt.cpp        # MqttNet — wrapper PubSubClient
├── topics/topics.cpp       # MQTT topic builder
├── codec/
│   ├── frames.h            # Struct definitions + CRC16 Modbus
│   └── codec.cpp           # Binary encode/decode
├── config/config.cpp       # NVS config storage
├── system/
│   ├── watchdog.h          # ESP32 task watchdog (15s)
│   └── logger.h            # Minimal serial logger
└── diagnostic/diagnostic.h # Reset reason, free heap, cpu temp
```

## Binary Frame Format

### TelemetryFrame (16 bytes)
| Offset | Type | Field |
|--------|------|-------|
| 0 | u8 | version=1 |
| 1 | u8 | msgType=0x01 |
| 2-3 | u16 | seq |
| 4-7 | u32 | ts |
| 8-9 | i16 | total (reg2) |
| 10-11 | i16 | code (reg5) |
| 12-13 | i16 | weight (reg114) |
| 14-15 | u16 | CRC16 Modbus |

**Topic:** `iot/data/mdcw/{area}/{prefix}/telemetry`

### HeartbeatFrame (31 bytes)
| Offset | Type | Field |
|--------|------|-------|
| 0 | u8 | version=1 |
| 1 | u8 | msgType=0x02 |
| 2-3 | u16 | seq |
| 4-7 | u32 | ts |
| 8-11 | u32 | uptime |
| 12-15 | u32 | heap |
| 16-19 | u32 | ipAddr |
| 20 | i8 | rssi |
| 21-22 | i16 | cpuTempX10 |
| 23 | u8 | resetReason |
| 24-25 | u16 | interval |
| 26 | u8 | fwMajor |
| 27 | u8 | fwMinor |
| 28-29 | u16 | CRC16 Modbus |

**Topic:** `iot/node/mdcw/{area}/{prefix}/heartbeat` (retained)

## Cara Pakai

1. **Credentials** — buat `platformio_local.ini`:
   ```ini
   [env:esp32-dev]
   build_flags =
       -D WIFI_SSID='"ssid"'
       -D WIFI_PASS='"pass"'
       -D MQTT_BROKER='"10.201.40.1"'
       -D MQTT_USER='"apps"'
       -D MQTT_PASS='"apps"'
       -D SERIAL_DEBUG
   ```
2. **Build + Upload**: `pio run -t upload`
3. **Monitor**: `pio device monitor`

## Topik MQTT

| Topic | Format | Retain |
|-------|--------|--------|
| `iot/data/mdcw/{area}/{prefix}/telemetry` | Binary 16B | ❌ |
| `iot/node/mdcw/{area}/{prefix}/heartbeat` | Binary 31B | ✅ |
| `cmd/ota` | JSON | — |
| `cmd/config` | JSON | — |

## Fitur

- ✅ Ring buffer offline (500 entry, ~5KB RAM)
- ✅ Auto-flush saat MQTT reconnect
- ✅ Watchdog 15s (di-feed setiap loop + OTA chunk)
- ✅ OTA rollback safety (ESP_OTA_IMG_PENDING_VERIFY → MQTT connect → cancel rollback)
- ✅ SHA256 verification opsional
- ✅ Heartbeat retained — subscriber baru langsung dapat status
