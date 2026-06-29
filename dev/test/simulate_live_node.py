import os
import paho.mqtt.client as mqtt
import time
import random
import json

# ==========================================
# Konfigurasi
# ==========================================
LOCAL_BROKER = os.getenv("MQTT_LOCAL_BROKER", "localhost")
LOCAL_PORT   = int(os.getenv("MQTT_LOCAL_PORT", 1883))

MQTT_USERNAME = os.getenv("MQTT_USERNAME", "apps")
MQTT_PASSWORD = os.getenv("MQTT_PASSWORD", "apps")

# ==========================================
# Node Suhu & Pintu yang disimulasikan
# (CK 3, dikirim CSV ke sensor/data/ingest)
# ==========================================
TEMP_NODES = [
    {"id": "ESP32_DEV_001", "ck": 3, "area": 10, "door_id": 101, "sensor_no": 1},
    {"id": "ESP32_DEV_002", "ck": 3, "area": 11, "door_id": 102, "sensor_no": 1},
]

# ==========================================
# MDCW Forming Nodes
# Topik: production/mdcw (JSON)
# Forming service lokal akan:
#   1. Simpan ke DB lokal (production_mdcw)
#   2. Forward ke Cloud MQTT → prod/mdcw → backend monitoring
#
# reg2  = total pack count (monotonically increasing per prefix)
# reg5  = status code: 41=OK, 25=Under, 73=Over, 8201=Metal, 9=Idle
# reg114= berat (dalam 10x gram, contoh: 8910 = 891.0g)
# prefix= nama line MDCW (harus cocok dengan mapping di cloud_forwarder.go)
# ==========================================
MDCW_LINES = [
    {
        "prefix":       "MDCW1 (UK)",
        "pack_count":   random.randint(100, 500),   # simulated counter awal
        "weight_range": (8710, 9520),                # OK range gram×10
        "weight_ok":    9115,                        # target weight
    },
    {
        "prefix":       "MDCW2 (Siomay)",
        "pack_count":   random.randint(100, 500),
        "weight_range": (7040, 7540),
        "weight_ok":    7290,
    },
    {
        "prefix":       "MDCW3 (Pentol)",
        "pack_count":   random.randint(100, 500),
        "weight_range": (5840, 6340),
        "weight_ok":    6090,
    },
    {
        "prefix":       "MDCW4 (AP)",
        "pack_count":   random.randint(100, 500),
        "weight_range": (14940, 15660),
        "weight_ok":    15300,
    },
    {
        "prefix":       "MDCW5 (ACIN)",
        "pack_count":   random.randint(100, 500),
        "weight_range": (10100, 10270),
        "weight_ok":    10185,
    },
    {
        "prefix":       "MDCW6 (Lumpia)",
        "pack_count":   random.randint(100, 500),
        "weight_range": (3080, 3340),
        "weight_ok":    3210,
    },
]

# Status code dan probabilitasnya (OK paling sering)
STATUS_CHOICES = [
    (41,   0.75),    # OK / Pass
    (25,   0.10),    # Underweight
    (73,   0.08),    # Overweight
    (8201, 0.05),    # Metal Detected
    (9,    0.02),    # Idle / Mati
]

def pick_status():
    """Pilih status code random berbobot."""
    rand = random.random()
    cumulative = 0.0
    for code, prob in STATUS_CHOICES:
        cumulative += prob
        if rand <= cumulative:
            return code
    return 41


def simulate_mdcw_payload(line: dict) -> dict:
    """Buat payload JSON MDCW sesuai format yang dibaca forming service."""
    status = pick_status()
    line["pack_count"] += 1   # increment counter per paket

    # Generate weight berdasarkan status
    lo, hi = line["weight_range"]
    if status == 41:    # OK — dalam range
        weight = random.randint(lo, hi)
    elif status == 25:  # Under
        weight = random.randint(lo - 500, lo - 1)
    elif status == 73:  # Over
        weight = random.randint(hi + 1, hi + 500)
    elif status == 8201:  # Metal — weight bisa apa saja
        weight = random.randint(lo, hi)
    else:               # Idle — weight 0
        weight = 0

    ts = time.strftime("%Y-%m-%d %H:%M:%S")

    return {
        "ts":     ts,
        "reg2":   line["pack_count"],
        "reg5":   status,
        "reg114": weight,
        "prefix": line["prefix"],
    }


def status_label(reg5: int) -> str:
    labels = {41: "OK [PASS]", 25: "UNDER [<]", 73: "OVER  [>]", 8201: "METAL [!]", 9: "IDLE  [-]"}
    return labels.get(reg5, f"?({reg5})")


def weight_display(reg114: int) -> str:
    if reg114 == 0:
        return "  ---  "
    return f"{reg114/10:.1f}g"


def simulate_full_live():
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2, "Full_Live_Simulator")
    client.username_pw_set(MQTT_USERNAME, MQTT_PASSWORD)

    connected = False

    def on_connect(c, userdata, flags, reason_code, properties):
        nonlocal connected
        if reason_code == 0:
            connected = True
            print(f"  [OK] Terkoneksi ke MQTT Broker {LOCAL_BROKER}:{LOCAL_PORT}")
        else:
            print(f"  [FAIL] Gagal koneksi, reason code: {reason_code}")

    def on_disconnect(c, userdata, flags, reason_code, properties):
        nonlocal connected
        connected = False
        print(f"  [WARN] Koneksi terputus (reason: {reason_code}). Mencoba reconnect...")

    client.on_connect    = on_connect
    client.on_disconnect = on_disconnect

    print("=" * 65)
    print("  FULL LIVE SIMULATOR -- Suhu + MDCW Forming")
    print("=" * 65)
    print(f"  Broker  : {LOCAL_BROKER}:{LOCAL_PORT}")
    print(f"  User    : {MQTT_USERNAME}")
    print(f"  Suhu    : {len(TEMP_NODES)} node(s) -> sensor/data/ingest (CSV)")
    print(f"  MDCW    : {len(MDCW_LINES)} line(s) -> production/mdcw (JSON)")
    print()
    print("  [ALUR SUHU]")
    print("   HW -> sensor/data/ingest (CSV) -> Servfor Forwarder")
    print("       -> sensor/data/forwarded (JSON) -> IoT-BE -> Dashboard Suhu")
    print()
    print("  [ALUR FORMING]")
    print("   HW -> production/mdcw (JSON) -> forming-app (servfor)")
    print("       -> DB lokal + CLOUD_MQTT (prod/mdcw) -> IoT-BE -> Dashboard Forming")
    print("=" * 65)
    print("  Tekan Ctrl+C untuk berhenti.\n")

    try:
        client.connect(LOCAL_BROKER, LOCAL_PORT, keepalive=60)
        client.loop_start()
        time.sleep(1.5)   # tunggu on_connect

        if not connected:
            print("  [!] Belum terkoneksi setelah 1.5s, tetap lanjut (auto-retry)...")

        cycle = 0
        while True:
            cycle += 1
            now = time.strftime("%Y-%m-%d %H:%M:%S")
            ts_iso = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())

            print(f"\n--- Siklus #{cycle:04d}  [{now}] -------------------------------------------")

            # -- A. Suhu & Pintu --
            print("  [SUHU/PINTU]")
            for node in TEMP_NODES:
                temp        = round(random.uniform(24.0, 30.0), 2)
                humi        = round(random.uniform(60.0, 75.0), 2)
                door_status = random.choice([0, 1])
                relay       = temp > 28.5

                # Status payload (OTA dashboard)
                client.publish(f"nodes/{node['id']}/status", json.dumps({
                    "status":  "online",
                    "model":   "TEMP|4",
                    "version": "v1.5.0-sim",
                    "ip":      f"192.168.1.{random.randint(10, 200)}",
                    "ck":      str(node["ck"]),
                    "area":    str(node["area"]),
                    "no":      str(node["sensor_no"]),
                }), retain=True)

                # Monitor payload
                client.publish(f"nodes/{node['id']}/monitor", json.dumps({
                    "ram_free_bytes": random.randint(150000, 220000),
                    "sd_ok":   True,
                    "relay":   relay,
                    "cur_t1":  temp,
                    "cur_p1":  1 if relay else 0,
                }))

                # Sensor data CSV -> sensor/data/ingest
                csv_data = (
                    f"CSV,CK,{node['ck']},AREA,{node['area']},"
                    f"TS,{ts_iso},M,{node['sensor_no']},"
                    f"{temp},{humi},D,{node['door_id']},{door_status}"
                )
                res = client.publish("sensor/data/ingest", csv_data)

                door_icon  = "OPEN" if door_status == 0 else "CLOSE"
                relay_icon = "ON" if relay else "OFF"
                status_pub = "OK" if res.rc == 0 else f"FAIL(rc={res.rc})"
                print(f"    {node['id']} Area={node['area']} "
                      f"T={temp}C H={humi}% Door={door_icon} Relay={relay_icon} | pub:{status_pub}")

            # ── B. MDCW Forming ──────────────────────────────────────────
            print("  [MDCW FORMING]")
            # Hanya kirim 2–4 line random per siklus (realistis)
            active_lines = random.sample(MDCW_LINES, k=random.randint(2, len(MDCW_LINES)))
            for line in active_lines:
                payload = simulate_mdcw_payload(line)
                json_str = json.dumps(payload)
                res = client.publish("production/mdcw", json_str)

                slabel  = status_label(payload["reg5"])
                wdisplay = weight_display(payload["reg114"])
                status_pub = "OK" if res.rc == 0 else f"FAIL(rc={res.rc})"
                print(f"    {line['prefix']:<18} pack#{payload['reg2']:>5} "
                      f"w={wdisplay:>8}  {slabel:<12} | pub:{status_pub}")

            time.sleep(10)

    except KeyboardInterrupt:
        print("\n\n  Menghentikan simulasi...")
        for node in TEMP_NODES:
            client.publish(f"nodes/{node['id']}/status",
                           json.dumps({"status": "offline"}), retain=True)
        print("  Semua node ditandai offline.")
    except Exception as e:
        print(f"  [ERR] {e}")
    finally:
        client.loop_stop()
        client.disconnect()
        print("  Simulator selesai. Goodbye!")


if __name__ == "__main__":
    simulate_full_live()
