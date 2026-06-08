# Ringkasan Sistem OTA & Monitoring (OTA Core)

Dokumen ini menjelaskan mekanisme kerja, struktur data, dan fungsionalitas sistem monitoring serta management perangkat (khususnya seri **TEMP**) pada environment ini.

## 1. Arsitektur Komunikasi
Sistem ini menggunakan protokol **MQTT** sebagai jalur komunikasi utama antara perangkat (ESP32) dan server (Go Backend).

*   **Broker**: EMQX (default).
*   **Arah Data**:
    *   **Device -> Server**: Update status, log, dan telemetri (Topik: `nodes/+/status`, `nodes/+/monitor`).
    *   **Server -> Device**: Perintah konfigurasi, OTA, dan reboot (Topik: `nodes/+/command`).

---

## 2. Model Khusus: TEMP (Temperature Sensor)
Model `TEMP` adalah basis untuk perangkat pemantau suhu. Ada beberapa varian (M1-M11) yang membedakan jumlah sensor suhu (DS18B20) dan sensor proximity.

### B. Varian Model M (Series)
Sistem menggunakan kategori model M1 s/d M11 untuk menentukan konfigurasi hardware (Relay, SD Card, dan Dotmatrix selalu tersedia di hampir semua model).

| Model | Sensor Suhu (DS18B20) | Proximity | Fitur Tambahan | Deskripsi |
| :--- | :---: | :---: | :--- | :--- |
| **M1** | 1 | - | Max Dotmatrix | Suhu Tunggal + Display |
| **M2** | Modbus | 1 | Max Dotmatrix | Modbus Module + Proximity |
| **M3** | 2 | 2 | Max Dotmatrix | 2 Suhu + 2 Proximity |
| **M4** | 3 | 1 | Max Dotmatrix | 3 Suhu + 1 Proximity |
| **M5** | 1 | 2 | Max Dotmatrix | 1 Suhu + 2 Proximity |
| **M6** | 1 | 1 | Max Dotmatrix | 1 Suhu + 1 Proximity |
| **M7** | - | 1 | - | Proximity + Relay/SD Only |
| **M8** | Modbus | - | Max Dotmatrix | Modbus Module Only |
| **M9** | 1 | - | - | Suhu + Relay/SD (Tanpa Display) |
| **M10** | 2 | 1 | Max Dotmatrix | 2 Suhu + 1 Proximity |
| **M11** | - | 2 | - | 2 Proximity + Relay/SD Only |

> *Catatan: Semua model di atas sudah dilengkapi dengan **Relay** dan **SD Card Module**.*

### C. Parameter Konfigurasi (Field)
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `ck` | Number | ID Central Kitchen (Lokasi Utama). |
| `area` | Number | ID Area spesifik dalam lokasi. |
| `interval` | Number | Jeda pengiriman data dalam milidetik (ms). |
| `min0` s/d `min4` | Float | Ambang batas bawah suhu untuk sensor ke-N. |
| `max0` s/d `max4` | Float | Ambang batas atas suhu untuk sensor ke-N. |
| `prox_nc0` s/d `prox_nc2` | Int | Logika sensor proximity (0: NO, 1: NC). |

### B. Contoh Pesan JSON (TEMP)

#### 1. Pesan Status (Dikirim Perangkat Secara Periodic)
Perangkat mengirimkan ini ke topik `nodes/[MAC_ID]/status` agar dashboard tahu kondisi alat.
```json
{
  "state": "online",
  "model": "TEMP",
  "ver": "1.2.0-M4",
  "ip": "192.168.1.50",
  "conf": {
    "ck": "1",
    "area": "10",
    "no": "5",
    "interval": 30000,
    "min0": -20.5,
    "max0": -15.0,
    "prox_nc0": 0
  }
}
```

#### 2. Pesan Monitor & RAM (Dikirim Perangkat)
Dikirim ke topik `nodes/[MAC_ID]/monitor` untuk pemantauan kesehatan hardware.
```json
{
  "ram_free_bytes": 124560,
  "sd_ok": true,
  "model": "TEMP",
  "prefix": "FRZ-01"
}
```

#### 3. Perintah Konfigurasi (Dari Server ke Perangkat)
Dikirim ke topik `nodes/[MAC_ID]/command` saat user menekan tombol "Save Config" di dashboard.
```json
{
  "cmd": "set_config",
  "ck": 1,
  "area": 12,
  "interval": 60000,
  "min0": -25.0,
  "max0": -10.5
}
```

---

## 3. Fitur Utama Sistem

### A. Over-The-Air (OTA) Update
Digunakan untuk memperbarui firmware tanpa akses fisik.
*   **JSON Command**:
    ```json
    {
      "cmd": "ota",
      "url": "http://server-ip:9999/files/firmware_v2.bin"
    }
    ```
*   **Cara Kerja**: Perangkat menerima URL, mengunduh file biner, melakukan flashing ke partisi kedua, lalu reboot.

### B. Remote Reboot
Memaksa perangkat restart jika terjadi error melalui MQTT.
*   **JSON Command**:
    ```json
    { "cmd": "reboot" }
    ```

### C. Konsol Log (Live Logs)
Perangkat mengirimkan log debug ke topik `nodes/[MAC_ID]/log`. Server menyimpan 10 baris terakhir untuk ditampilkan di dashboard sebagai alat debug cepat (fast troubleshooting).

---

## 4. Cara Kerja Konfigurasi (Config Workflow)

Sistem menggunakan alur **Event-Driven** untuk memperbarui pengaturan alat tanpa menyentuh fisik:

1.  **Input User**: Melalui Dashboard UI (Form Model-Specific).
2.  **API Backend**: Web mengirim POST Request ke `/api/config` dengan payload data sensor.
3.  **MQTT Publish**: Server mengirim perintah ke topik `nodes/[MAC]/command`.
4.  **Perangkat (Client)**: ESP32 menerima, memparsing JSON, menyimpan ke NVS/EEPROM, dan membalas dengan status terbaru.

## 5. Daftar Contoh JSON Per Model (MQTT Command)

Berikut adalah muatan (payload) JSON lengkap yang diharapkan dikirim server ke alat (Topik: `nodes/+/command`) untuk setiap varian:

### Model M1 (1 Suhu)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 30000, "min0": -20.0, "max0": -10.0 }
```

### Model M2 (Modbus + 1 Proximity)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 60000, "prox_nc0": 1 }
```

### Model M3 (2 Suhu + 2 Proximity)
```json
{
  "cmd": "set_config",
  "ck": 1, "area": 1, "interval": 30000,
  "min0": -20.0, "max0": -10.0,
  "min1": 2.0, "max1": 8.0,
  "prox_nc0": 1, "prox_nc1": 1
}
```

### Model M4 (3 Suhu + 1 Proximity)
```json
{
  "cmd": "set_config",
  "ck": 1, "area": 1, "interval": 30000,
  "min0": -20.0, "max0": -10.0,
  "min1": -20.0, "max1": -10.0,
  "min2": -20.0, "max2": -10.0,
  "prox_nc0": 0
}
```

### Model M5 (1 Suhu + 2 Proximity)
```json
{
  "cmd": "set_config",
  "ck": 1, "area": 1, "interval": 30000,
  "min0": -18.0, "max0": -12.0,
  "prox_nc0": 1, "prox_nc1": 1
}
```

### Model M6 (1 Suhu + 1 Proximity)
```json
{
  "cmd": "set_config",
  "ck": 1, "area": 1, "interval": 30000,
  "min0": -18.0, "max0": -12.0,
  "prox_nc0": 1
}
```

### Model M7 (1 Proximity Only)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 30000, "prox_nc0": 1 }
```

### Model M8 (Modbus Only)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 60000 }
```

### Model M9 (1 Suhu Only)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 30000, "min0": -20.0, "max0": -10.0 }
```

### Model M10 (2 Suhu + 1 Proximity)
```json
{
  "cmd": "set_config",
  "ck": 1, "area": 1, "interval": 30000,
  "min0": -20.0, "max0": -10.0,
  "min1": -15.0, "max1": -5.0,
  "prox_nc0": 1
}
```

### Model M11 (2 Proximity Only)
```json
{ "cmd": "set_config", "ck": 1, "area": 1, "interval": 15000, "prox_nc0": 1, "prox_nc1": 1 }
```
