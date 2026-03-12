#!/usr/bin/env python3
"""
IoT Device Simulator for OTA Dashboard
Mensimulasikan ESP32/IoT device yang mengirim data ke MQTT broker
"""

import paho.mqtt.client as mqtt
import json
import time
import random
import argparse
from datetime import datetime

class IoTDeviceSimulator:
    def __init__(self, broker_host, broker_port=1883, username="apps", password="apps"):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.username = username
        self.password = password
        self.client = None
        self.devices = []
        
    def generate_mac(self):
        """Generate random MAC address"""
        return ''.join([random.choice('0123456789ABCDEF') for _ in range(12)])
    
    def create_device(self, model="TEMP", prefix="NODE", mac=None):
        """Create a simulated device"""
        if mac is None:
            mac = self.generate_mac()
        
        device = {
            'model': model,
            'prefix': prefix,
            'mac': mac,
            'node_id': f"{prefix}-{mac}",
            'ram_free': random.randint(50000, 200000),
            'sd_ok': random.choice([True, False]),
            'status': 'running',
            'ck': f"CK{random.randint(1, 10)}",
            'area': f"Area{random.randint(1, 5)}",
            'no': str(random.randint(1, 100))
        }
        
        self.devices.append(device)
        print(f"✅ Created device: {device['node_id']} (Model: {model})")
        return device
    
    def on_connect(self, client, userdata, flags, rc):
        """Callback when connected to MQTT broker"""
        if rc == 0:
            print(f"🔗 Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
            # Subscribe to command topics for all devices
            for device in self.devices:
                topic = f"nodes/{device['node_id']}/command"
                client.subscribe(topic)
                print(f"📡 Subscribed to: {topic}")
        else:
            print(f"❌ Connection failed with code {rc}")
    
    def on_message(self, client, userdata, msg):
        """Callback when receiving MQTT message"""
        try:
            payload = json.loads(msg.payload.decode())
            print(f"\n📨 Received command on {msg.topic}")
            print(f"   Payload: {payload}")
            
            # Extract node_id from topic
            parts = msg.topic.split('/')
            if len(parts) >= 2:
                node_id = parts[1]
                
                # Find device
                device = next((d for d in self.devices if d['node_id'] == node_id), None)
                if device:
                    cmd = payload.get('cmd', '')
                    
                    if cmd == 'ota':
                        url = payload.get('url', '')
                        print(f"   🚀 OTA Update requested: {url}")
                        self.send_log(device, f"Starting OTA update from {url}")
                        time.sleep(1)
                        self.send_log(device, "Downloading firmware...")
                        time.sleep(1)
                        self.send_log(device, "OTA update completed!")
                        
                    elif cmd == 'set_threshold':
                        min_temp = payload.get('min', 0)
                        max_temp = payload.get('max', 0)
                        ck = payload.get('ck', '')
                        area = payload.get('area', '')
                        no = payload.get('no', '')
                        print(f"   ⚙️ Config: Min={min_temp}, Max={max_temp}, CK={ck}, Area={area}, No={no}")
                        device['ck'] = ck
                        device['area'] = area
                        device['no'] = no
                        self.send_log(device, f"Config updated: CK={ck}, Area={area}, No={no}")
                        
                    elif cmd == 'set_config':
                        prefix = payload.get('prefix', '')
                        print(f"   ⚙️ MDCW Config: Prefix={prefix}")
                        old_prefix = device['prefix']
                        device['prefix'] = prefix
                        old_node_id = device['node_id']
                        device['node_id'] = f"{prefix}-{device['mac']}"
                        print(f"   🔄 Node ID changed: {old_node_id} -> {device['node_id']}")
                        self.send_log(device, f"Prefix changed to {prefix}, restarting...")
                        
        except Exception as e:
            print(f"❌ Error processing message: {e}")
    
    def send_status(self, device):
        """Send status message"""
        topic = f"nodes/{device['node_id']}/status"
        payload = {
            'state': device['status'],
            'model': device['model'],
            'prefix': device['prefix']
        }
        
        self.client.publish(topic, json.dumps(payload), retain=True)
        print(f"📤 Status sent: {device['node_id']} -> {device['status']}")
    
    def send_monitor(self, device):
        """Send monitor message"""
        topic = f"nodes/{device['node_id']}/monitor"
        
        # Simulate RAM fluctuation
        device['ram_free'] += random.randint(-5000, 5000)
        device['ram_free'] = max(30000, min(250000, device['ram_free']))
        
        payload = {
            'ram_free_bytes': device['ram_free'],
            'sd_ok': device['sd_ok'],
            'model': device['model'],
            'prefix': device['prefix']
        }
        
        self.client.publish(topic, json.dumps(payload), retain=True)
        print(f"📊 Monitor sent: {device['node_id']} -> RAM: {device['ram_free']} bytes")
    
    def send_log(self, device, message):
        """Send log message"""
        topic = f"nodes/{device['node_id']}/log"
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        log_msg = f"[{timestamp}] {message}"
        
        self.client.publish(topic, log_msg)
        print(f"📝 Log sent: {device['node_id']} -> {message}")
    
    def connect(self):
        """Connect to MQTT broker"""
        self.client = mqtt.Client(client_id=f"simulator-{random.randint(1000, 9999)}")
        self.client.username_pw_set(self.username, self.password)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        try:
            print(f"🔌 Connecting to {self.broker_host}:{self.broker_port}...")
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()
            time.sleep(2)  # Wait for connection
            return True
        except Exception as e:
            print(f"❌ Connection error: {e}")
            return False
    
    def run(self, interval=5, duration=None):
        """Run simulation"""
        print(f"\n🚀 Starting simulation with {len(self.devices)} device(s)")
        print(f"   Update interval: {interval} seconds")
        if duration:
            print(f"   Duration: {duration} seconds")
        print("\n" + "="*60)
        
        start_time = time.time()
        iteration = 0
        
        try:
            while True:
                iteration += 1
                print(f"\n⏱️  Iteration #{iteration} - {datetime.now().strftime('%H:%M:%S')}")
                print("-" * 60)
                
                for device in self.devices:
                    # Send status and monitor data
                    self.send_status(device)
                    time.sleep(0.2)
                    self.send_monitor(device)
                    time.sleep(0.2)
                    
                    # Randomly send logs
                    if random.random() < 0.3:  # 30% chance
                        messages = [
                            "System running normally",
                            "Sensor reading completed",
                            "WiFi signal: -65 dBm",
                            "Temperature check OK",
                            "Heartbeat sent"
                        ]
                        self.send_log(device, random.choice(messages))
                        time.sleep(0.2)
                
                # Check duration
                if duration and (time.time() - start_time) >= duration:
                    print(f"\n⏰ Duration reached ({duration}s), stopping simulation")
                    break
                
                # Wait for next iteration
                print(f"\n💤 Waiting {interval} seconds...")
                time.sleep(interval)
                
        except KeyboardInterrupt:
            print("\n\n⛔ Simulation stopped by user")
        finally:
            self.disconnect()
    
    def simulate_offline(self, device_index=0, duration=15):
        """Simulate device going offline"""
        if device_index < len(self.devices):
            device = self.devices[device_index]
            print(f"\n⚠️  Simulating offline for {device['node_id']} ({duration}s)")
            device['status'] = 'offline'
            self.send_status(device)
            time.sleep(duration)
            device['status'] = 'running'
            self.send_status(device)
            print(f"✅ {device['node_id']} back online")
    
    def disconnect(self):
        """Disconnect from MQTT broker"""
        if self.client:
            print("\n🔌 Disconnecting from MQTT broker...")
            self.client.loop_stop()
            self.client.disconnect()
            print("✅ Disconnected")


def main():
    parser = argparse.ArgumentParser(description='IoT Device Simulator for OTA Dashboard')
    parser.add_argument('--broker', default='localhost', help='MQTT broker host (default: localhost)')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port (default: 1883)')
    parser.add_argument('--user', default='apps', help='MQTT username (default: apps)')
    parser.add_argument('--password', default='apps', help='MQTT password (default: apps)')
    parser.add_argument('--devices', type=int, default=3, help='Number of devices to simulate (default: 3)')
    parser.add_argument('--interval', type=int, default=5, help='Update interval in seconds (default: 5)')
    parser.add_argument('--duration', type=int, help='Simulation duration in seconds (optional)')
    parser.add_argument('--model', default='TEMP', choices=['TEMP', 'MDCW'], help='Device model (default: TEMP)')
    
    args = parser.parse_args()
    
    # Create simulator
    sim = IoTDeviceSimulator(
        broker_host=args.broker,
        broker_port=args.port,
        username=args.user,
        password=args.password
    )
    
    # Create devices
    print("\n" + "="*60)
    print("🎮 IoT Device Simulator")
    print("="*60)
    
    for i in range(args.devices):
        prefix = f"NODE{chr(65+i)}" if args.model == 'TEMP' else f"MDCW{i+1}"
        sim.create_device(model=args.model, prefix=prefix)
    
    # Connect and run
    if sim.connect():
        # Send initial status for all devices
        time.sleep(1)
        for device in sim.devices:
            sim.send_status(device)
            sim.send_monitor(device)
            sim.send_log(device, "Device initialized")
            time.sleep(0.5)
        
        # Run simulation
        sim.run(interval=args.interval, duration=args.duration)
    else:
        print("❌ Failed to connect to MQTT broker")


if __name__ == "__main__":
    main()
