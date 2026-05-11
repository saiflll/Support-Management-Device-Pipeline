import os
import paho.mqtt.client as mqtt
import time
import random
import json

# ==========================================
# Konfigurasi
# ==========================================
LOCAL_BROKER = os.getenv("MQTT_LOCAL_BROKER", "localhost")
LOCAL_PORT = int(os.getenv("MQTT_LOCAL_PORT", 1883))

# Daftar Node yang disimulasikan
# [FIX] sensor_no ESP32_DEV_002 diubah 2->1: Area 11 hanya daftarkan SensorNo=1 di config.go TempThresholds
NODES = [
    {"id": "ESP32_DEV_001", "ck": 3, "area": 10, "door_id": 101, "sensor_no": 1},
    {"id": "ESP32_DEV_002", "ck": 3, "area": 11, "door_id": 102, "sensor_no": 1},
]

# ==========================================
# MQTT Credentials (ambil dari env atau default sesuai .env servfor)
# [FIX] Default 'apps'/'apps' sesuai EMQX_AUTH__USER__1 di docker-compose.yml servfor
# ==========================================
MQTT_USERNAME = os.getenv("MQTT_USERNAME", "apps")
MQTT_PASSWORD = os.getenv("MQTT_PASSWORD", "apps")


def simulate_full_live():
    # Gunakan Callback API v1 untuk kompatibilitas paho-mqtt 2.x
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION1, "Full_Live_Simulator")

    # [FIX] Set credentials sebelum connect
    client.username_pw_set(MQTT_USERNAME, MQTT_PASSWORD)

    try:
        client.connect(LOCAL_BROKER, LOCAL_PORT)
        print(f"=== FULL LIVE SIMULATOR STARTED ===")
        print(f"Connecting to Broker: {LOCAL_BROKER}:{LOCAL_PORT}")
        print(f"Simulating {len(NODES)} nodes...")
        print(f"[ALUR] HW(sim) -> sensor/data/ingest (CSV) -> Servfor Forwarder -> sensor/data/forwarded (JSON) -> BE -> FE")

        while True:
            for node in NODES:
                node_id = node["id"]
                timestamp = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
                readable_time = time.strftime("%Y-%m-%d %H:%M:%S")

                temp = round(random.uniform(24.0, 30.0), 2)
                humi = round(random.uniform(60.0, 75.0), 2)
                door_status = random.choice([0, 1])
                relay_status = temp > 28.5

                # 1. Kirim STATUS (Dashboard OTA)
                status_payload = {
                    "status": "online",
                    "model": "TEMP|4",
                    "version": "v1.5.0-live",
                    "ip": f"192.168.1.{random.randint(10, 200)}",
                    "ck": str(node["ck"]),
                    "area": str(node["area"]),
                    "no": str(node["sensor_no"])
                }
                client.publish(f"nodes/{node_id}/status", json.dumps(status_payload), retain=True)

                # 2. Kirim MONITOR (RAM, SD, Alarm Relay)
                monitor_payload = {
                    "ram_free_bytes": random.randint(150000, 220000),
                    "sd_ok": True,
                    "relay": relay_status,
                    "cur_t1": temp,
                    "cur_p1": 1 if relay_status else 0
                }
                client.publish(f"nodes/{node_id}/monitor", json.dumps(monitor_payload))

                # 3. Kirim DATA SENSOR sebagai CSV ke sensor/data/ingest
                #    Forwarder (servfor) yang akan parse CSV, agregasi, lalu forward ke
                #    sensor/data/forwarded → iot-suhu-be menerima dan simpan ke DB
                csv_data = f"CSV,CK,{node['ck']},AREA,{node['area']},TS,{timestamp},M,{node['sensor_no']},{temp},{humi},D,{node['door_id']},{door_status}"
                client.publish("sensor/data/ingest", csv_data)

                # 4. Kirim LOG (Dashboard Logs)
                logs = [
                    f"Sensor read success: T={temp}C, H={humi}%",
                    f"MQTT Publish OK -> sensor/data/ingest",
                    "Battery Voltage: 3.95V",
                    "Heartbeat OK"
                ]
                client.publish(f"nodes/{node_id}/log", random.choice(logs))

                print(f"[{readable_time}] Node {node_id} (Area {node['area']}, Sensor {node['sensor_no']}): "
                      f"T={temp}°C H={humi}% Door={door_status} | CSV->ingest ✓")

            time.sleep(10)  # HW baca tiap 10 detik

    except KeyboardInterrupt:
        print("\nStopping simulation...")
        for node in NODES:
            client.publish(f"nodes/{node['id']}/status", json.dumps({"status": "offline"}), retain=True)
        print("All nodes marked as offline.")
    except Exception as e:
        print(f"Error: {e}")
    finally:
        client.disconnect()

if __name__ == "__main__":
    simulate_full_live()
