# System Documentation & Protocol Specification

Dokumentasi lengkap mengenai format JSON untuk **Sensor Data Ingestion** (Forwarder) dan **Device Actuator/Management** (OTA Dashboard).

---

# PART 1: SENSOR DATA INGESTION (FORWARDER)

Format ini digunakan oleh device (atau gateway) untuk mengirimkan data sensor (Suhu, RH, Pintu) ke sistem Forwarder. Data ini yang akan diproses, di-buffer, dan diteruskan ke broker tujuan (Cloud/DB).

**Topik MQTT Default**: `sensor/data/ingest` (Bisa dikonfigurasi per device)

## ✅ Valid Data Formats

### 1. Format Lengkap (Standard)
Format paling standar yang disarankan.
```json
{
  "ck": 5,                  // [REQUIRED] ID Central Kitchen (Integer)
  "area": 20,               // [REQUIRED] ID Area (Integer)
  "door": [                 // [REQUIRED] Array Pintu (Boleh kosong [])
    {
      "doorid": 101,        // ID Pintu (Integer)
      "value": 0            // 0: Terbuka, 1: Tertutup
    }
  ],
  "temp": [                 // [REQUIRED] Array Suhu (Min 1 item)
    {
      "no": 1,              // Nomor Urut Sensor (Integer)
      "ts": "2026-02-11T11:37:01.422922+07:00", // RFC3339Nano Timezone
      "temp": 18.06,        // Suhu Float
      "rh": 54.1            // [OPTIONAL] Humidity Float (boleh null)
    }
  ]
}
```

### 2. Format Minimal (Tanpa RH & Pintu)
```json
{
  "ck": 3,
  "area": 14,
  "door": [],
  "temp": [
    {
      "no": 8,
      "ts": "2026-02-11T11:52:51.301271+07:00",
      "temp": -41.44,
      "rh": null
    }
  ]
}
```

### 3. Format Multi-Sensor
```json
{
  "ck": 5,
  "area": 20,
  "door": [
    { "doorid": 1, "value": 0 },
    { "doorid": 2, "value": 1 }
  ],
  "temp": [
    {
      "no": 1,
      "ts": "2026-02-11T11:37:01+07:00",
      "temp": 22.5,
      "rh": 65.3
    },
    {
      "no": 2,
      "ts": "2026-02-11T11:37:02+07:00",
      "temp": 23.1,
      "rh": 64.8
    }
  ]
}
```

### 4. Batching (Array)
Forwarder akan mengumpulkan data-data di atas dan mengirimkannya ke Cloud dalam bentuk Array untuk efisiensi.
```json
[
  { "ck": 5, "area": 20, "temp": [...], "door": [...] },
  { "ck": 5, "area": 20, "temp": [...], "door": [...] }
]
```

---

# PART 2: DEVICE ACTUATOR & MANAGEMENT (OTA)

Format ini digunakan untuk komunikasi antara **Dashboard OTA** dan **Device** (ESP32).
Digunakan untuk konfigurasi remote, reboot, dan update firmware.

**Topik MQTT Device**:
*   Subscribe: `nodes/{NODE_ID}/command` (Menerima perintah)
*   Publish: `nodes/{NODE_ID}/status` (Melaporkan status & config saat ini)
*   Publish: `nodes/{NODE_ID}/monitor` (Data live untuk dashboard)

## 📡 Actuator Commands (Downlink)

Dashboard mengirim JSON ini ke toplik `nodes/{NODE_ID}/command`.

### 1. Set Configuration (`set_config`)
Mengubah parameter device secara remote. Field yang dikirim tergantung Model device.

**Model: TEMP (Umum), M1-M11**
```json
{
  "cmd": "set_config",      // [REQUIRED] Command ID
  "ck": 5,                  // [OPTIONAL] Set CK ID
  "area": 20,               // [OPTIONAL] Set Area ID
  "no": 1,                  // [OPTIONAL] Set Node Number
  "interval": 1000,         // [OPTIONAL] Interval kirim data (ms)
  "delay": 500,             // [OPTIONAL] Delay baca sensor (ms)
  
  // Calibration / Offsets
  "min": -2.0,              // Offset/Min Temp Value
  "max": 2.0,               // Offset/Max Temp Value
  
  // Specific Ranges (M10, M11, etc)
  "min0": 0.0, "max0": 0.0, // Range T1
  "min1": 0.0, "max1": 0.0, // Range T2
  "min2": 0.0, "max2": 0.0, // Range T3
  
  // Proximity Logic
  "prox_nc0": 1,            // 1=NC (Normally Closed), 0=NO
  "prox_nc1": 0,
  
  // Humidity
  "min_rh": 0.0, 
  "max_rh": 100.0
}
```

**Model: MDCW / V1 / V2 (Timbangan)**
```json
{
  "cmd": "set_config",
  "prefix": "MDCW_01",      // Kode Prefix Alat
  "interval": 500,
  "prox_nc0": 1
}
```

**Model: TROLI**
```json
{
  "cmd": "set_config",
  "app_mode": "A",          // "A" (IN/OUT) atau "B" (Product Checking)
  "trans": "IN",            // "IN" atau "OUT" (Mode A)
  "pass_code": "100209"     // Target Product Code (Mode B)
}
```

### 2. OTA Update (`ota`)
Memerintahkan device untuk download firmware baru.
```json
{
  "cmd": "ota",
  "url": "http://192.168.x.x:9999/files/firmware_v2.bin"
}
```

### 3. Reboot Device (`reboot`)
Memerintahkan device untuk restart.
```json
{
  "cmd": "reboot"
}
```

---

## 📡 Device Reporting (Uplink)

Device mengirim JSON ini ke dashboard.

### 1. Status Report (`nodes/{NODE_ID}/status`)
Dikirim saat boot atau saat konfigurasi berubah (Retained Message).
```json
{
  "status": "online",
  "ip": "192.168.1.50",
  "model": "M1",
  "version": "1.0.2-M1",
  "updated": "2026-02-11 13:00:00", // Waktu terakhir update
  "ram_free_bytes": 145000,
  "sd_ok": true,                    // Status SD Card
  
  // Current Config Values (Mirror dari setting)
  "ck": "5",
  "area": "20",
  "no": "1",
  "interval": 1000,
  "prox_nc0": 1,
  "min0": 0.5
}
```

### 2. Live Monitor (`nodes/{NODE_ID}/monitor`)
Dikirim secara periodik untuk update data live di dashboard (mirip Status tapi lebih ringan/sering).
```json
{
  "ram_free_bytes": 144500,
  "status": "online"
  // Bisa ditambahkan data sensor live jika diperlukan dashboard
}
```

---

## 📝 Catatan Implementasi

1.  **Timezone**: Selalu gunakan WIB (UTC+7) atau sertakan offset `+07:00` dalam timestamp.
2.  **Field Validation**: Forwarder akan membuang extra field tak dikenal, tapi Actuator (ESP32) harus memparsing JSON dengan toleransi (abaikan field tak dikenal).
3.  **Topic Forwarding**: Forwarder otomatis menghapus field `topic` dari payload sensor sebelum diteruskan ke cloud (sesuai Format 4).
4.  **Error Handling**: Jika JSON `set_config` salah tipe data (misal string dikirim ke int), ESP32 sebaiknya mengabaikan field tersebut dan tidak crash.