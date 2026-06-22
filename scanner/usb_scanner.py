import os
import sys
import time
import threading
import paho.mqtt.client as mqtt
import keyboard
import serial
import serial.tools.list_ports

MQTT_BROKER = "mqtt-dashboard.com"
MQTT_TOPIC  = "Syalannn"
MQTT_PORT   = 1883
MAX_SCANNER_DURATION = 0.30

mqtt_client = None
hid_buffer = ""
hid_start_time = None

def init_mqtt():
    global mqtt_client
    try:
        mqtt_client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
        mqtt_client.connect(MQTT_BROKER, MQTT_PORT, 60)
        mqtt_client.loop_start()
        print(f"[+] MQTT: Connected to -> {MQTT_BROKER}:{MQTT_PORT}")
    except Exception as e:
        print(f"[!] MQTT WARN: Broker offline ({e}). Running on Local Log Mode.")
        mqtt_client = None

def forward_to_mqtt(payload, source_type):
    global mqtt_client
    ts = time.strftime("%H:%M:%S")
    clean_payload = payload.strip()
    
    if not clean_payload:
        return
        
    print(f"[{ts}] [IN DATA] [{source_type}] -> {clean_payload}")
    
    if mqtt_client:
        res = mqtt_client.publish(MQTT_TOPIC, clean_payload, qos=1)
        if res.rc == mqtt.MQTT_ERR_SUCCESS:
            print(f"[{ts}] [OUT MQTT] : [{MQTT_TOPIC}]")
        else:
            print(f"[!] MQTT ERR: Failed to send packet.")
    else:
        print(f"[{ts}] [LOCAL LOG] Broker unreachable.")
def start_hid_sniffer():
    print("[+] HID ENGINE: Global Keyboard Hook Active with Burst Filter...")
    
    def on_key(event):
        global hid_buffer, hid_start_time
        
        if event.event_type == keyboard.KEY_DOWN:
            if not hid_buffer:
                hid_start_time = time.time()
                
            if event.name == 'enter':
                if hid_buffer:
                    duration = time.time() - hid_start_time
                    if duration <= MAX_SCANNER_DURATION:
                        forward_to_mqtt(hid_buffer, "HID_SCANNER")
                    else:
                        ts = time.strftime("%H:%M:%S")
                        print(f"[{ts}] [-] FILTERED: Speed: {duration:.2f}s (Data: {hid_buffer} Too Slow Hooman)")
                    hid_buffer = ""
                    hid_start_time = None
            elif len(event.name) == 1:
                hid_buffer += event.name
    keyboard.hook(on_key)
def serial_reader_worker(port_name):
    ts = time.strftime("%H:%M:%S")
    print(f"[{ts}] [+] VCP ENGINE: Locking serial stream on {port_name}...")
    try:
        ser = serial.Serial(port_name, baudrate=9600, timeout=1)
        print(f"[{ts}] [+] VCP LOCKED: Exclusive stream active on {port_name}")
        
        while ser.is_open:
            if ser.in_waiting > 0:
                raw_line = ser.readline().decode('utf-8', errors='ignore')
                if raw_line:
                    forward_to_mqtt(raw_line, "VCP_PORT")
            time.sleep(0.05)
    except Exception:
        pass 

def start_vcp_sniffer():
    print("[+] VCP ENGINE: Monitoring Virtual COM Ports...")
    active_threads = {}
    
    while True:
        ports = list(serial.tools.list_ports.comports())
        current_ports = [port.device for port in ports]
        
        for port in current_ports:
            if port not in active_threads or not active_threads[port].is_alive():
                t = threading.Thread(target=serial_reader_worker, args=(port,), daemon=True)
                t.start()
                active_threads[port] = t
                
        time.sleep(2)

def main():
    os.system('cls' if os.name == 'nt' else 'clear')
    print("=======================================================================")
    print("  HYBRID AUTOMATION GATEWAY ENGINE: HID SPEED BURST & VCP COM   ")
    print("===+===+===+===+===+===+===+===+ RENNN +===+===+===+===+===+===+===+===")
    init_mqtt()
    start_hid_sniffer()
    vcp_thread = threading.Thread(target=start_vcp_sniffer, daemon=True)
    vcp_thread.start()
    
    print("\n[+] SYSTEM READY: Pipeline active 24/7.")
    print("[i] Terminal status: FREE FOCUS (Boleh di-minimize).")
    print("[i] Menunggu trigger data karton... (Tekan Ctrl+C untuk stop)\n")
    
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n[-] SYSTEM: Gateway terminated by operator.")
    finally:
        keyboard.unhook_all()
        if mqtt_client:
            mqtt_client.loop_stop()
            mqtt_client.disconnect()
        print("[+] STATUS: Clean exit. All ports released.")

if __name__ == "__main__":
    main()
