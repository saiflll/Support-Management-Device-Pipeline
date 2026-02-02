# Panduan Lengkap Perangkat & Pemetaan Sensor (M1 - M11)

Dokumen ini memberikan panduan fisik dan teknis untuk seluruh varian perangkat Seri M, termasuk pemetaan sensor ke dalam struktur JSON.

## 1. Konsep Pemetaan Sensor
Setiap perangkat memiliki slot fisik yang harus dipetakan ke field JSON yang tepat agar dashboard menampilkan data dengan benar.

| Komponen Fisik | Field JSON | Keterangan |
| :--- | :--- | :--- |
| Sensor Suhu 1 | `min0` / `max0` | Sensor utama (biasanya DS18B20). |
| Sensor Suhu 2 | `min1` / `max1` | Tersedia di model M3, M4, M10. |
| Sensor Suhu 3 | `min2` / `max2` | Tersedia di model M4. |
| Proximity 1 | `prox_nc0` | Input digital sensor pintu 1. |
| Proximity 2 | `prox_nc1` | Input digital sensor pintu 2. |

---

## 2. Detil Konfigurasi Per Model

### [M1] Suhu Tunggal
*   **Hardware**: 1x DS18B20 + Dotmatrix Display.
*   **JSON Config**: `{"min0": -20.0, "max0": -15.0}`
*   **Penggunaan**: Freezer/Chiller standar.

### [M2] Modbus + Proximity
*   **Hardware**: Modbus Module + 1x Proximity + Dotmatrix.
*   **JSON Config**: `{"prox_nc0": 0}` (0: NO, 1: NC)
*   **Penggunaan**: Integrasi sensor sht30  via Modbus.

### [M3] 2 Suhu + 2 Proximity
*   **Hardware**: 2x DS18B20 + 2x Proximity + Dotmatrix.
*   **JSON Config**: `{"min0": -20, "max0": -15, "min1": 2, "max1": 8, "prox_nc0": 0, "prox_nc1": 0}`
*   **Penggunaan**: 2 area dan pintu pintu dengan pemantauan suhu terpisah.

### [M4] 3 Suhu + 1 Proximity
*   **Hardware**: 3x DS18B20 + 1x Proximity + Dotmatrix.
*   **JSON Config**: `{"min0": -18, "min1": -18, "min2": -18, "prox_nc0": 0}`
*   **Penggunaan**: Cold Storage besar dengan 3 titik ukur suhu.

### [M5] 1 Suhu + 2 Proximity
*   **Hardware**: 1x DS18B20 + 2x Proximity + Dotmatrix.
*   **JSON Config**: `{"min0": -20, "max0": -10, "prox_nc0": 0, "prox_nc1": 0}`
*   **Penggunaan**: Pemantauan 1 suhu dan 2 pintu (misal: pintu depan & belakang).

### [M6] 1 Suhu + 1 Proximity
*   **Hardware**: 1x DS18B20 + 1x Proximity + Dotmatrix.
*   **JSON Config**: `{"min0": -18, "prox_nc0": 0}`
*   **Penggunaan**: Chiller standar dengan sensor pintu.

### [M7] Proximity Only (1 Channel)
*   **Hardware**: 1x Proximity + Relay + SD Card.
*   **JSON Config**: `{"prox_nc0": 0}`
*   **Penggunaan**: Hanya mendeteksi status pintu (tanpa suhu & display).

### [M8] Modbus Only
*   **Hardware**: Modbus Module + Dotmatrix.
*   **JSON Config**: `{"ck": 1, "area": 1, "interval": 60000}`
*   **Penggunaan**: Display data Modbus berupa suhu dan temperatur.

### [M9] Suhu Only (Tanpa Display)
*   **Hardware**: 1x DS18B20 + Relay + SD Card.
*   **JSON Config**: `{"min0": -20}`
*   **Penggunaan**: Data logger suhu tersembunyi/tanpa layar.

### [M10] 2 Suhu + 1 Proximity
*   **Hardware**: 2x DS18B20 + 1x Proximity + Dotmatrix.
*   **JSON Config**: `{"min0": -18, "min1": -18, "prox_nc0": 0}`
*   **Penggunaan**: 2 Titik suhu + 1 deteksi pintu.

### [M11] Proximity Only (2 Channel)
*   **Hardware**: 2x Proximity + Relay + SD Card.
*   **JSON Config**: `{"prox_nc0": 0, "prox_nc1": 0}`
*   **Penggunaan**: Pemantauan 2 status pintu.

---

## 3. Catatan Penting
1.  **Interval**: Disarankan minimal **30.000 ms** (30 detik) untuk menjaga umur sensor dan stabilitas jaringan.
2.  **Relay**: Semua perangkat dapat memicu Relay secara internal berdasarkan logika `min/max` .
3.  **SD Card**: Digunakan sebagai backup jika koneksi WiFi/MQTT terputus (Offline Logging).
