import json
import time
import random
import requests
import paho.mqtt.client as mqtt
from datetime import datetime
from threading import Thread

# --- MEGDEV ENVIRONMENT CONFIG ---
MQTT_BROKER = "localhost" 
MQTT_PORT = 1883
MQTT_USER = "apps"
MQTT_PASS = "apps"

OTA_API_URL = "http://localhost:9999"

# Helper to mimic MAC address
def get_mock_mac(id_int):
    return f"FAD0B1C%02X" % id_int

class BaseDevice:
    def __init__(self, node_id, model):
        self.node_id = node_id
        self.model = model
        self.client = self._setup_mqtt()
        
    def _setup_mqtt(self):
        # PENTING: client_id HARUS SAMA dengan node_id agar webhook & MQTT sinkron
        try:
            c = mqtt.Client(mqtt.CallbackAPIVersion.VERSION1, client_id=self.node_id)
        except:
            c = mqtt.Client(self.node_id)
            
        c.username_pw_set(MQTT_USER, MQTT_PASS)
        c.on_message = self.on_message
        return c

    def on_message(self, client, userdata, msg):
        print(f"\n[{self.model}] 📥 Command In [{self.node_id}]: {msg.payload.decode()}")

    def connect(self):
        try:
            self.client.connect(MQTT_BROKER, MQTT_PORT)
            self.client.subscribe(f"nodes/{self.node_id}/command")
            self.client.loop_start()
            print(f"[{self.model}] 🚀 Online: {self.node_id}")
        except Exception as e:
            print(f"[{self.model}] ❌ Broker Connection Failed: {e}")

    def publish_telemetry(self, status_payload, monitor_payload):
        # Pastikan model dikirimkan di kedua payload agar sinkron
        self.client.publish(f"nodes/{self.node_id}/status", json.dumps(status_payload))
        self.client.publish(f"nodes/{self.node_id}/monitor", json.dumps(monitor_payload))

    def publish_log(self, msg):
        self.client.publish(f"nodes/{self.node_id}/log", msg)

    def stop(self):
        self.client.loop_stop()
        self.client.disconnect()

# 1. TROLI (DOUGHT) SIMULATOR
class TroliDevice(BaseDevice):
    def __init__(self, mac):
        node_id = f"troliOUT{mac}"
        super().__init__(node_id, "TROLI")
        
    def run_loop(self):
        while True:
            status = {"state": "Running", "model": "TROLI"}
            monitor = {
                "ram_free_bytes": random.randint(100000, 200000), 
                "model": "TROLI",
                "sd_ok": True
            }
            self.publish_telemetry(status, monitor)
            
            # Simulasi Log Berkala
            self.publish_log(f"Troli Engine OK - Pack Count: {random.randint(50, 200)}")
            
            # Simulasi Scan Barcode (Random)
            if random.random() > 0.8:
                barcode = f"888{random.randint(1111,9999)}"
                scan = {
                    "mode": "troli", "barcode": barcode, "success": True,
                    "product_name": "ROTI TAWAR MEGDEV",
                    "tanggal": datetime.now().strftime("%Y-%m-%d"),
                    "batch": random.randint(1,5), "shift": 1
                }
                self.client.publish(f"nodes/{self.node_id}/scan", json.dumps(scan))
                print(f"[TROLI] ✅ Scan Transmitted: {barcode}")
                
            time.sleep(5)

# 2. MDCW (FORMING) SIMULATOR
class MdcwDevice(BaseDevice):
    def __init__(self, prefix, mac):
        self.prefix = prefix
        node_id = f"{prefix}-{mac}"
        super().__init__(node_id, "MDCW")
        
    def run_loop(self):
        while True:
            status = {"state": "running", "model": "MDCW", "prefix": self.prefix}
            monitor = {
                "ram_free_bytes": random.randint(80000, 150000), 
                "model": "MDCW", "prefix": self.prefix
            }
            self.publish_telemetry(status, monitor)
            
            # Data Produksi ke Topic Khusus
            prod_data = {
                "ts": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                "reg2": random.randint(2000, 3000), "reg5": 41,
                "reg114": random.randint(450, 465), # Berat
                "prefix": self.prefix
            }
            self.client.publish("production/mdcw", json.dumps(prod_data))
            time.sleep(5)

# 3. TEMP (SHT30) SIMULATOR
class TempDevice(BaseDevice):
    def __init__(self, ck, area, no, mac):
        self.ck, self.area, self.no = ck, area, no
        self.prefix = f"{ck}-{area}-{no}"
        node_id = f"{self.prefix}{mac}"
        super().__init__(node_id, "TEMP")
        
    def run_loop(self):
        while True:
            status = {"state": "Running", "model": "TEMP", "prefix": self.prefix}
            monitor = {
                "ram_free_bytes": random.randint(40000, 70000), 
                "sd_ok": True, "model": "TEMP", "prefix": self.prefix
            }
            self.publish_telemetry(status, monitor)
            
            # Ingest Data ke Forwarder
            ingest_data = [{
                "ck": self.ck, "area": self.area, "node": self.node_id, "door": [],
                "temp": [{
                    "no": self.no, "ts": datetime.now().isoformat() + "+07:00",
                    "temp": round(random.uniform(18.0, 22.0), 2),
                    "rh": round(random.uniform(50.0, 60.0), 2),
                    "relay": 0, "door": 1
                }]
            }]
            self.client.publish("sensor/data/ingest", json.dumps(ingest_data))
            time.sleep(5)

def start_sim(device):
    device.connect()
    device.run_loop()

if __name__ == "__main__":
    print("\n" + "="*50)
    print("  MEGDEV SIMULATOR v2.2 - FIXING SYNC")
    print("  Ensuring ClientID matches NodeID")
    print("="*50)
    
    # Inisialisasi perangkat dengan ID dan MAC unik
    devices = [
        TroliDevice(get_mock_mac(11)),
        MdcwDevice("LINE-B", get_mock_mac(22)),
        TempDevice(5, 20, 2, get_mock_mac(33))
    ]
    
    threads = []
    for d in devices:
        t = Thread(target=start_sim, args=(d,), daemon=True)
        t.start()
        threads.append(t)
        
    print(f"\n[SYSTEM] � 3 Perangkat terhubung ke Broker: {MQTT_BROKER}")
    print("[SYSTEM] � Silakan refresh Dashboard untuk melihat node tunggal (No Dedupe Conflict)")
    
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n[Megdev] Stopping...")