import time
import json
import random
import paho.mqtt.client as mqtt

# Konfigurasi MQTT
MQTT_BROKER = "127.0.0.1"  # Sesuaikan dengan IP Server/Broker jika tidak di lokal
MQTT_PORT = 1883
MQTT_USER = "apps"
MQTT_PASS = "apps"
MQTT_TOPIC = "production/mdcw"

# Daftar target per prefix (dengan berat dasar dan variasinya)
# Base Weight dalam satuan 0.1g (misal: 9000 = 900.0g)
PREFIXES = {
    "MDCW1 (UK)":     {"base": 9100,  "var": 600},
    "MDCW2 (SIOMAY)": {"base": 7300,  "var": 400},
    "MDCW3 (PENTOL)": {"base": 6100,  "var": 400},
    "MDCW4 (AP)":     {"base": 15300, "var": 500},
    "MDCW5 (ACIN)":   {"base": 10200, "var": 150},
    "MDCW6 (LUMPIA)": {"base": 3200,  "var": 200},
    "MDCW8":          {"base": 8000,  "var": 2000},
    "MDCW9":          {"base": 4000,  "var": 3000},
}

# Inisialisasi reg2 (Pack Count) dengan angka acak awal
pack_counts = {p: random.randint(1000, 5000) for p in PREFIXES}

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("Berhasil terhubung ke Broker MQTT!")
    else:
        print(f"Gagal terhubung, return code {rc}")

# Fungsi status yang mengikuti toleransi berat di date_filter.go
def get_status(weight, prefix):
    p = prefix.upper()
    if "MDCW1" in p:
        if weight < 8710: return 25   # Under
        elif weight > 9520: return 73 # Over
        else: return 41               # OK
    elif "MDCW2" in p:
        if weight < 7040: return 25
        elif weight > 7540: return 73
        else: return 41
    elif "MDCW3" in p:
        if weight < 5840: return 25
        elif weight > 6340: return 73
        else: return 41
    elif "MDCW4" in p:
        if weight < 14940: return 25
        elif weight > 15660: return 73
        else: return 41
    elif "MDCW5" in p:
        if weight < 10100: return 25
        elif weight > 10270: return 73
        else: return 41
    elif "MDCW6" in p:
        if weight < 3080: return 25
        elif weight > 3340: return 73
        else: return 41
    else:
        # Acak untuk MDCW lain yang tidak masuk daftar rules
        return random.choice([41, 41, 41, 25, 73]) 

# Setup MQTT Client
client = mqtt.Client()
client.username_pw_set(MQTT_USER, MQTT_PASS)
client.on_connect = on_connect

print(f"Menyambungkan ke {MQTT_BROKER}:{MQTT_PORT} ...")
client.connect(MQTT_BROKER, MQTT_PORT, 60)
client.loop_start()

try:
    print("Mulai mengirim data simulasi. Tekan CTRL+C untuk berhenti.")
    while True:
        # Pilih satu mesin secara acak setiap iterasi
        prefix = random.choice(list(PREFIXES.keys()))
        config = PREFIXES[prefix]
        
        # Kalkulasi berat acak disekitar nilai dasar
        weight = random.randint(config["base"] - config["var"], config["base"] + config["var"])
        
        # 3% Kemungkinan mendapat status METAL (reg5 = 8201)
        is_metal = random.random() < 0.03
        if is_metal:
            status = 8201
        else:
            status = get_status(weight, prefix)

        # Tambah pack count
        pack_counts[prefix] += 1
        
        payload = {
            "ts": int(time.time() * 1000),
            "reg2": pack_counts[prefix],
            "reg5": status,
            "reg114": weight,
            "prefix": prefix
        }
        
        msg = json.dumps(payload)
        client.publish(MQTT_TOPIC, msg)
        print(f"Terkirim -> {msg}")
            
        # Jeda pengiriman diatur acak agar terlihat natural seperti mesin asli (0.5 - 2 detik)
        time.sleep(random.uniform(0.5, 2.0))

except KeyboardInterrupt:
    print("\nMenghentikan simulasi...")
    client.loop_stop()
    client.disconnect()
