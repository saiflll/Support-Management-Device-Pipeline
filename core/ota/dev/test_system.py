import paho.mqtt.client as mqtt
import json
import time
import sys

# --- CONFIGURATION ---
MQTT_BROKER = "localhost" # Ganti ke IP server EMQX
MQTT_PORT = 1883
NODE_ID = "SIM-TEST-01"
MODEL = "MDCW-DYNAMIC-TEST"

print(f"Starting Simulation for {NODE_ID}...")

def on_connect(client, userdata, flags, rc):
    print(f"Connected with result code {rc}")
    client.subscribe(f"nodes/{NODE_ID}/command")
    
    # Send Initial Status
    status = {
        "status": "online",
        "ip": "10.0.0.50",
        "model": MODEL,
        "version": "1.2.3-DEBUG",
        "interval": 15000,
        "prefix": "SIM",
        "temp_limit": 25.5,
        "alarm_enabled": True
    }
    client.publish(f"nodes/{NODE_ID}/status", json.dumps(status), retain=True)
    print("Sent initial status with dynamic config fields.")

def on_message(client, userdata, msg):
    print(f"\n[MQTT] Received Command on {msg.topic}")
    try:
        payload = json.loads(msg.payload.decode())
        cmd = payload.get("cmd")
        print(f"[CMD] {cmd.upper()} requested.")
        
        if cmd == "get_config":
            # Simulate response to config request
            client.publish(f"nodes/{NODE_ID}/logs", "[LOG] Fetching current config for dashboard...")
            status = {
                "status": "online",
                "interval": 15000,
                "prefix": "SIM",
                "temp_limit": 25.5,
                "alarm_enabled": True,
                "wifi_rssi": -65
            }
            client.publish(f"nodes/{NODE_ID}/status", json.dumps(status))
            print("[SIM] Sent fresh config to broker.")
            
        elif cmd == "set_config":
            print(f"[SIM] Applying new settings: {payload}")
            client.publish(f"nodes/{NODE_ID}/logs", "[LOG] Saving new config to NVS...")
            time.sleep(1)
            client.publish(f"nodes/{NODE_ID}/logs", "[LOG] Config saved. System rebooting...")
            
        elif cmd == "reboot":
            client.publish(f"nodes/{NODE_ID}/logs", "[LOG] Remote reboot command received.")
            print("[SIM] Rebooting...")

        elif cmd == "ota":
            client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] Starting OTA from {payload.get('url')}...")
            for i in range(0, 101, 50):
                time.sleep(0.5)
                client.publish(f"nodes/{NODE_ID}/logs", f"[LOG] OTA Progress: {i}%")
            client.publish(f"nodes/{NODE_ID}/logs", "[LOG] OTA Success.")

    except Exception as e:
        print(f"Error parsing command: {e}")

client = mqtt.Client()
client.on_connect = on_connect
client.on_message = on_message

try:
    client.connect(MQTT_BROKER, MQTT_PORT, 60)
except Exception as e:
    print(f"Could not connect to MQTT broker: {e}")
    sys.exit(1)

# Run for 20 seconds to allow for manual testing or just verification
print("Node is now active. Send commands from Dashboard to see them here.")
client.loop_start()

start_time = time.time()
while time.time() - start_time < 20: 
    # Periodically publish monitor data
    mon = {"ram": 120000, "temp": 24.5 + (time.time() % 2)}
    client.publish(f"nodes/{NODE_ID}/monitor", json.dumps(mon))
    time.sleep(5)

client.loop_stop()
print("\nSimulation ended.")
