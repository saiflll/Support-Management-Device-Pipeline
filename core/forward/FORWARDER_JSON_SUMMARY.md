# Dokumentasi Sistem Forwarder (Data Aggregator)

Sistem `Forwarder` berfungsi sebagai jembatan yang mengumpulkan data telemetri (suhu & pintu) dari perangkat lokal, menyimpannya sementara dalam buffer, lalu mengirimkannya secara massal (batch) ke broker MQTT Publik setiap 8 menit atau saat ukuran data mencapai 50KB.

## 1. Topik MQTT
*   **Topic Ingest (Lokal)**: `sensor/data/ingest` (Perangkat mengirim data ke sini).
*   **Topic Public (Forward)**: Ditentukan via environment `MQTT_TOPIC_PUB` (Data diteruskan ke sini dalam bentuk array).

---

## 2. Struktur JSON Telemetri (AreaData)

Perangkat diharapkan mengirimkan data dalam format `AreaData`. Format ini mendukung pengiriman data suhu (`temp`) dan data pintu/proximity (`door`) secara bersamaan.

### Objek Utama: `AreaData`
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `ck` | Integer | ID Central Kitchen. |
| `area` | Integer | ID Area (terdaftar di konfigurasi sensor). |
| `temp` | Array | Daftar objek `TempData`. |
| `door` | Array | Daftar objek `DoorData`. |

### Sub-Objek: `TempData`
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `no` | Integer | Nomor urut sensor (0 s/d 4). |
| `ts` | String | Timestamp format RFC3339Nano (opsional, jika kosong akan diisi server). |
| `temp` | Float | Nilai suhu yang dibaca. |
| `rh` | Float | Nilai kelembaban (opsional, bisa null). |

### Sub-Objek: `DoorData`
| Field | Tipe | Deskripsi |
| :--- | :--- | :--- |
| `doorid` | Integer | ID unik pintu (harus sesuai database pintu). |
| `value` | Integer | Status pintu (0: Tutup, 1: Terbuka). |

---

## 3. Contoh JSON yang Diharapkan

### A. Data Suhu Saja
```json
{
  "ck": 1,
  "area": 10,
  "temp": [
    {
      "no": 0,
      "ts": "2023-10-27T10:00:00Z",
      "temp": -18.5
    }
  ],
  "door": []
}
```

### B. Data Pintu Saja
```json
{
  "ck": 1,
  "area": 10,
  "temp": [],
  "door": [
    {
      "doorid": 101,
      "value": 0
    }
  ]
}
```

### C. Data Gabungan (Suhu + Pintu + Kelembaban)
```json
{
  "ck": 1,
  "area": 10,
  "temp": [
    {
      "no": 0,
      "temp": -18.5,
      "rh": 65.0
    },
    {
      "no": 1,
      "temp": 4.2
    }
  ],
  "door": [
    {
      "doorid": 101,
      "value": 0
    },
    {
      "doorid": 102,
      "value": 1
    }
  ]
}
```

---

## 4. Mekanisme Forwarding
1.  **Agregasi**: Setiap pesan yang masuk ke server lokal akan dimasukkan ke `buffer`.
2.  **Interval**: Setiap **8 menit**, server akan membungkus semua data di buffer menjadi satu **Array JSON**.
3.  **Pengiriman**: Array tersebut dipublikasikan ke broker publik.

**Contoh Hasil Forwarding (di MQTT Publik):**
```json
[
  { "ck": 1, "area": 10, "temp": [...], "door": [...] },
  { "ck": 1, "area": 12, "temp": [...], "door": [...] },
  ...
]
```

---

## 5. Fitur Pendukung
*   **Telegram Alert**: Jika suhu melebihi ambang batas atau pintu terbuka terlalu lama, sistem otomatis mengirim pesan ke Telegram Group.
*   **Database Archiver**: Data yang masuk juga disimpan ke database lokal dan diarsip secara otomatis.
