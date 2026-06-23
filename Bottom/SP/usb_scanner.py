import os
import time
import threading
import paho.mqtt.client as mqtt
import keyboard
import serial
import serial.tools.list_ports

MQTT_BROKER = "172.20.100.11"
MQTT_TOPIC  = "SP_data"
MQTT_PORT   = 1883
MQTT_USER   = "SP_Account"
MQTT_PASS   = "pestaporaabadi"
MAX_SCANNER_DURATION = 0.30

mqtt_client = None
hid_buffer = ""
hid_start_time = None
SESSION_ID = ""

def get_session_id():
    global SESSION_ID
    while True:
        raw = input("  Masukkan Session ID (huruf & angka): ").strip()
        if raw.isalnum() and len(raw) > 0:
            SESSION_ID = raw
            print(f"  [+] Session ID terdaftar: {SESSION_ID}\n")
            break
        else:
            print("  [!] ID tidak valid. Gunakan huruf dan angka saja, tanpa spasi.")

def get_timestamp_gmt7():
    gmt7_time = time.gmtime(time.time() + 7 * 3600)
    return time.strftime("%Y-%m-%dT%H:%M:%S+07:00", gmt7_time)

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
    global mqtt_client, SESSION_ID
    clean_payload = payload.strip()
    if not clean_payload:
        return
    ts = get_timestamp_gmt7()
    mqtt_payload = f"{SESSION_ID}&{clean_payload}&{ts}"
    print(f"[{ts}] [IN DATA] [{source_type}] -> {clean_payload}")
    if mqtt_client:
        res = mqtt_client.publish(MQTT_TOPIC, mqtt_payload, qos=1)
        if res.rc == mqtt.MQTT_ERR_SUCCESS:
            print(f"[{ts}] [OUT MQTT] : [{MQTT_TOPIC}] -> {mqtt_payload}")
        else:
            print(f"[!] MQTT ERR: Failed to send packet.")
    else:
        print(f"[{ts}] [LOCAL LOG] -> {mqtt_payload}")

def start_hid_sniffer():
    print("[+] HID ENGINE: Global Keyboard Hook Active with Burst Filter...")
    def on_key(event):
        global hid_buffer, hid_start_time
        if event.event_type == keyboard.KEY_DOWN:
            if not hid_buffer:
                hid_start_time = time.time()
            if event.name == "enter":
                if hid_buffer:
                    duration = time.time() - hid_start_time
                    if duration <= MAX_SCANNER_DURATION:
                        forward_to_mqtt(hid_buffer, "HID_SCANNER")
                    else:
                        ts = get_timestamp_gmt7()
                        print(f"[{ts}] [-] FILTERED: Speed: {duration:.2f}s (Data: {hid_buffer} Too Slow Hooman)")
                    hid_buffer = ""
                    hid_start_time = None
            elif len(event.name) == 1:
                hid_buffer += event.name
    keyboard.hook(on_key)

def serial_reader_worker(port_name):
    ts = get_timestamp_gmt7()
    print(f"[{ts}] [+] VCP ENGINE: Locking serial stream on {port_name}...")
    try:
        ser = serial.Serial(port_name, baudrate=9600, timeout=1)
        print(f"[{ts}] [+] VCP LOCKED: Exclusive stream active on {port_name}")
        while ser.is_open:
            if ser.in_waiting > 0:
                raw_line = ser.readline().decode("utf-8", errors="ignore")
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
    os.system("cls" if os.name == "nt" else "clear")
    print("========================")
    print(" HID  & VCP Auto Detection ")
    print("===+ ITDT WEST 2025 +===")
    print()
    get_session_id()
    init_mqtt()
    start_hid_sniffer()
    vcp_thread = threading.Thread(target=start_vcp_sniffer, daemon=True)
    vcp_thread.start()
    print("\n[+] SYSTEM READY: Pipeline active ^-^.")
    print("[i] Boleh di-minimize, Menunggu trigger data...")
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