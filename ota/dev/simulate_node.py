import paho.mqtt.client as mqtt
import json
import time

# --- CONFIGURATION ---
MQTT_BROKER = "localhost" # Ganti ke IP server EMQX jika perlu
MQTT_PORT = 1883
NODE_ID = "SIM-MDCW-001"
MODEL = "MDCW-DYNAMIC"

# State Simulasi Node
config = {
    "node_prefix": "SIM-",
    "interval": 10,
    "min_t1": 2.0,
    "max_t1": 8.0,
    "pass_code": "1234",
    "ck": "Jakarta",
    "area": "Warehouse-A"
}

def on_connect(client, userdata, flags, rc):
    print(f"Node tertubung dengan kode: {rc}")
    # Subscribe ke topic command
    client.subscribe(f"nodes/{NODE_ID}/command")
    # Publish status awal
    publish_status(client)

def on_message(client, userdata, msg):
    print(f"Command diterima di topic {msg.topic}")
    try:
        payload = json.loads(msg.payload.decode())
        cmd = payload.get("cmd")
        
        if cmd == "get_config":
            print("Menerima request get_config. Mengirim status...")
            publish_status(client)
            
        elif cmd == "set_config":
            print(f"Menerima set_config: {payload}")
            # Update local config (exclude cmd)
            for k, v in payload.items():
                if k != "cmd":
                    config[k] = v
            print("Konfigurasi diupdate. Melakukan simulasi reboot...")
            client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] Config updated, rebooting...")
            time.sleep(1)
            publish_status(client) # Simulasi hidup lagi
            
        elif cmd == "reboot":
            print("Menerima perintah reboot.")
            client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] System rebooting...")
            time.sleep(2)
            publish_status(client)
            
        elif cmd == "ota":
            url = payload.get("url")
            print(f"Menerima perintah OTA ke {url}")
            client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] Starting OTA from {url}...")
            for i in range(0, 101, 20):
                time.sleep(0.5)
                client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] OTA Progress: {i}%")
            client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] OTA Success, restarting...")
            time.sleep(1)
            publish_status(client)

    except Exception as e:
        print(f"Gagal memproses message: {e}")

def publish_status(client):
    # Gabungkan metadata status dengan config
    status_msg = {
        "status": "online",
        "ip": "192.168.1.100",
        "ram": 154000,
        "model": MODEL,
        "version": "2.0.0-SIM",
        **config # Masukkan semua field config secara dinamis
    }
    client.publish(f"nodes/{NODE_ID}/status", json.dumps(status_msg), retain=True)
    print(f"Status dikirim: {status_msg}")

client = mqtt.Client(NODE_ID)
client.on_connect = on_connect
client.on_message = on_message

print(f"Simulasi Node {NODE_ID} berjalan...")
client.connect(MQTT_BROKER, MQTT_PORT, 60)

# Loop untuk simulasi data monitor rutin
last_status = 0
try:
    while True:
        client.loop(timeout=1.0)
        now = time.time()
        
        # Kirim data monitor setiap 5 detik
        if now - last_status > 5:
            monitor_data = {
                "t1": 4.5,
                "t2": 5.1,
                "p1": 0,
                "relay": False,
                "ram_free": 142000
            }
            client.publish(f"nodes/{NODE_ID}/monitor", json.dumps(monitor_data))
            last_status = now
except KeyboardInterrupt:
    print("Simulasi dihentikan.")
