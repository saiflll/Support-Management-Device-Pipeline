# Google Sheets Setup untuk Forming App

## 🚀 Cara Setup Google Sheets Integration

### 1. Buat Service Account di Google Cloud Console

1. Buka [Google Cloud Console](https://console.cloud.google.com/)
2. Buat project baru atau pilih project existing
3. Enable **Google Sheets API**:

   - Navigation Menu → APIs & Services → Library
   - Cari "Google Sheets API"
   - Klik Enable

4. Buat Service Account:

   - Navigation Menu → APIs & Services → Credentials
   - Click "Create Credentials" → Service Account
   - Isi nama (contoh: `forming-app-sheets`)
   - Klik "Create and Continue"
   - Skip role selection (klik Continue)
   - Klik Done

5. Buat JSON Key:
   - Klik service account yang baru dibuat
   - Tab "Keys" → Add Key → Create new key
   - Pilih JSON
   - **Download file JSON** (misal: `forming-credentials.json`)

### 2. Convert Credentials ke Base64

**Windows (PowerShell):**

```powershell
$content = Get-Content -Path "forming-credentials.json" -Raw
$bytes = [System.Text.Encoding]::UTF8.GetBytes($content)
$base64 = [Convert]::ToBase64String($bytes)
$base64 | Set-Clipboard
Write-Host "Base64 copied to clipboard!"
```

Atau pakai command dari Go app:

```bash
go run . setup-sheets forming-credentials.json
```

### 3. Setup Environment Variables

Edit file `.env.local`:

```bash
# Google Sheets Configuration
GOOGLE_SHEETS_CREDENTIALS=eyJAeXBlIjoic2VydmljZV9hY2NvdW50... (paste base64 here)
GOOGLE_SPREADSHEET_ID=1ABC... (your spreadsheet ID)
GOOGLE_SHEET_NAME=Production Data
```

**Cara dapat Spreadsheet ID:**

- Buka spreadsheet di Google Sheets
- Lihat URL: `https://docs.google.com/spreadsheets/d/[SPREADSHEET_ID]/edit`
- Copy bagian `[SPREADSHEET_ID]`

### 4. Share Spreadsheet dengan Service Account

1. Buka file `forming-credentials.json`
2. Copy email di field `client_email` (contoh: `forming-app-sheets@project.iam.gserviceaccount.com`)
3. Buka Google Spreadsheet yang ingin Anda gunakan
4. Klik **Share** button
5. Paste email service account
6. Pilih permission: **Editor**
7. Klik Send

### 5. Test Connection

Jalankan aplikasi:

```bash
go run main.go
```

Cek log, harusnya muncul:

```
✅ Google Sheets API initialized successfully
✅ Google Sheets headers created
```

Kirim data ke MQTT, dan data otomatis muncul di Google Spreadsheet!

## 📊 Format Data di Spreadsheet

| Timestamp           | Prefix | Pack Count | Status Code | Weight (g) | Status |
| ------------------- | ------ | ---------- | ----------- | ---------- | ------ |
| 2025-12-10 10:30:00 | LINE-1 | 150        | 1           | 250        | OK     |
| 2025-12-10 10:30:05 | LINE-2 | 151        | 2           | 230        | Under  |

## ⚙️ Konfigurasi Optional

### Ganti Nama Sheet

Default sheet name: `Production Data`

Untuk custom nama, edit `.env.local`:

```bash
GOOGLE_SHEET_NAME=CustomSheetName
```

### Disable Google Sheets

Jika tidak mau pakai Sheets, cukup hapus atau comment environment variables:

```bash
# GOOGLE_SHEETS_CREDENTIALS=...
# GOOGLE_SPREADSHEET_ID=...
```

Aplikasi akan tetap jalan normal tanpa export ke Sheets.

## 🔥 Troubleshooting

### Error: "credentials file not found"

- Pastikan path file JSON benar
- Check permission read file

### Error: "The caller does not have permission"

- Pastikan spreadsheet sudah di-share dengan service account email
- Check permission minimal: Editor

### Error: "API not enabled"

- Google Sheets API harus di-enable di Cloud Console
- Navigation Menu → APIs & Services → Library → Enable Google Sheets API

### Data tidak muncul di Sheets

- Check log aplikasi untuk error messages
- Verify `GOOGLE_SPREADSHEET_ID` benar
- Check nama sheet (`GOOGLE_SHEET_NAME`) sesuai dengan tab di spreadsheet

## 📝 Notes

- **Async Export**: Export ke Sheets berjalan async (non-blocking), jadi tidak memperlambat insert ke PostgreSQL
- **Auto-Retry**: Jika export gagal, akan di-log tapi tidak crash aplikasi
- **Rate Limits**: Google Sheets API punya rate limit. Untuk production dengan volume tinggi, consider batching
- **Free Tier**: Google Sheets API free untuk 60 requests/minute per project

## 🔐 Security Best Practices

1. **JANGAN commit** file `forming-credentials.json` ke Git
2. Add ke `.gitignore`:
   ```
   *credentials*.json
   .env.local
   ```
3. Untuk production, gunakan Google Cloud Secret Manager instead of base64 env var
4. Rotate service account keys secara berkala

---

Selamat! Data produksi sekarang otomatis sync ke Google Spreadsheet! 📊✨
