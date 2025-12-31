# Model Configuration System - Developer Guide

## 📋 Overview

Sistem konfigurasi model yang baru memungkinkan penambahan model device baru dengan mudah tanpa perlu mengubah banyak kode. Cukup tambahkan definisi model di `modelRegistry`.

## 🏗️ Architecture

### Model Registry
Semua model didefinisikan di `modelRegistry` (map) dengan struktur:

```go
var modelRegistry = map[string]ModelConfig{
    "MODEL_NAME": {
        Name:        "MODEL_NAME",
        DisplayName: "Human Readable Name",
        Command:     "mqtt_command_name",
        Fields: []Field{
            // Field definitions
        },
    },
}
```

### Field Types
Setiap field memiliki properties:
- `Name`: Field name dalam JSON
- `Label`: Label untuk ditampilkan di UI
- `Type`: "number", "text", "select"
- `Required`: true/false
- `Placeholder`: Hint text (optional)
- `Min`, `Max`, `Step`: Untuk number type (optional)
- `Options`: Untuk select type (optional)

## ✅ Existing Models

### 1. TEMP (Temperature Sensor)
```go
"TEMP": {
    Name:        "TEMP",
    DisplayName: "Temperature Sensor",
    Command:     "set_threshold",
    Fields: []Field{
        {Name: "min", Label: "Min Temp (°C)", Type: "number", Required: true, Step: floatPtr(0.1)},
        {Name: "max", Label: "Max Temp (°C)", Type: "number", Required: true, Step: floatPtr(0.1)},
        {Name: "ck", Label: "CK", Type: "text", Required: true},
        {Name: "area", Label: "Area", Type: "text", Required: true},
        {Name: "no", Label: "No", Type: "text", Required: true},
    },
},
```

**MQTT Command:**
```json
{
  "cmd": "set_threshold",
  "min": 20.5,
  "max": 30.0,
  "ck": "CK01",
  "area": "Area1",
  "no": "001"
}
```

### 2. MDCW
```go
"MDCW": {
    Name:        "MDCW",
    DisplayName: "MDCW Device",
    Command:     "set_config",
    Fields: []Field{
        {Name: "prefix", Label: "Prefix (Node Name)", Type: "text", Required: true, Placeholder: "e.g., NODE_A"},
    },
},
```

**MQTT Command:**
```json
{
  "cmd": "set_config",
  "prefix": "NODE_A"
}
```

## 🚀 How to Add New Model

### Step 1: Add Model to Registry

Edit `ota/main.go`, tambahkan model baru di `modelRegistry`:

```go
var modelRegistry = map[string]ModelConfig{
    // ... existing models ...
    
    "DOOR": {
        Name:        "DOOR",
        DisplayName: "Door Sensor",
        Command:     "set_door_config",
        Fields: []Field{
            {
                Name:     "zone",
                Label:    "Zone",
                Type:     "text",
                Required: true,
            },
            {
                Name:     "alert_delay",
                Label:    "Alert Delay (seconds)",
                Type:     "number",
                Required: true,
                Min:      floatPtr(0),
                Max:      floatPtr(60),
                Step:     floatPtr(1),
            },
            {
                Name:     "sensitivity",
                Label:    "Sensitivity",
                Type:     "select",
                Required: true,
                Options:  []string{"low", "medium", "high"},
            },
        },
    },
}
```

### Step 2: Update NodeInfo (if needed)

Jika model baru memerlukan field khusus untuk disimpan di NodeInfo:

```go
type NodeInfo struct {
    Status       string
    RamFreeBytes int64
    SD_OK        *bool
    Ck           string
    Area         string
    No           string
    Updated      string
    Model        string
    Prefix       string
    
    // Add new fields for DOOR model
    Zone         string   `json:"zone,omitempty"`
    AlertDelay   int      `json:"alert_delay,omitempty"`
    Sensitivity  string   `json:"sensitivity,omitempty"`
    
    Logs         []string
}
```

### Step 3: Update Config Handler (if needed)

Jika perlu custom logic untuk menyimpan field:

```go
// In /config endpoint
switch field.Name {
case "ck":
    info.Ck = fmt.Sprint(value)
case "area":
    info.Area = fmt.Sprint(value)
case "no":
    info.No = fmt.Sprint(value)
case "prefix":
    info.Prefix = fmt.Sprint(value)
// Add new cases
case "zone":
    info.Zone = fmt.Sprint(value)
case "alert_delay":
    if v, ok := value.(float64); ok {
        info.AlertDelay = int(v)
    }
case "sensitivity":
    info.Sensitivity = fmt.Sprint(value)
}
```

### Step 4: Device Implementation

Device harus:

1. **Send model in status message:**
```json
{
  "status": "online",
  "model": "DOOR",
  "zone": "Zone1",
  "alert_delay": 5,
  "sensitivity": "medium"
}
```

2. **Handle config command:**
```cpp
// ESP32 example
void handleMQTTCommand(String payload) {
    JsonDocument doc;
    deserializeJson(doc, payload);
    
    String cmd = doc["cmd"];
    
    if (cmd == "set_door_config") {
        String zone = doc["zone"];
        int alertDelay = doc["alert_delay"];
        String sensitivity = doc["sensitivity"];
        
        // Save to EEPROM/preferences
        preferences.putString("zone", zone);
        preferences.putInt("alert_delay", alertDelay);
        preferences.putString("sensitivity", sensitivity);
        
        // Apply config
        applyDoorConfig();
    }
}
```

## 📊 API Endpoints

### Get All Models
```bash
GET /api/models
```

Response:
```json
{
  "TEMP": {
    "name": "TEMP",
    "display_name": "Temperature Sensor",
    "command": "set_threshold",
    "fields": [...]
  },
  "MDCW": {...},
  "DOOR": {...}
}
```

### Get Specific Model
```bash
GET /api/models/DOOR
```

Response:
```json
{
  "name": "DOOR",
  "display_name": "Door Sensor",
  "command": "set_door_config",
  "fields": [
    {
      "name": "zone",
      "label": "Zone",
      "type": "text",
      "required": true
    },
    ...
  ]
}
```

## 🎨 Frontend Integration

Frontend akan otomatis render form berdasarkan model config:

```javascript
// Fetch model config
fetch(`/api/models/${model}`)
    .then(res => res.json())
    .then(config => {
        // Render fields dynamically
        config.fields.forEach(field => {
            if (field.type === 'number') {
                // Render number input
            } else if (field.type === 'text') {
                // Render text input
            } else if (field.type === 'select') {
                // Render select dropdown
            }
        });
    });
```

## 📝 Example: Adding HUMIDITY Model

```go
"HUMIDITY": {
    Name:        "HUMIDITY",
    DisplayName: "Humidity Sensor",
    Command:     "set_humidity_threshold",
    Fields: []Field{
        {
            Name:     "min_humidity",
            Label:    "Min Humidity (%)",
            Type:     "number",
            Required: true,
            Min:      floatPtr(0),
            Max:      floatPtr(100),
            Step:     floatPtr(0.1),
        },
        {
            Name:     "max_humidity",
            Label:    "Max Humidity (%)",
            Type:     "number",
            Required: true,
            Min:      floatPtr(0),
            Max:      floatPtr(100),
            Step:     floatPtr(0.1),
        },
        {
            Name:        "location",
            Label:       "Location",
            Type:        "text",
            Required:    true,
            Placeholder: "e.g., Warehouse A",
        },
        {
            Name:     "alert_mode",
            Label:    "Alert Mode",
            Type:     "select",
            Required: true,
            Options:  []string{"email", "telegram", "both"},
        },
    },
},
```

## ✅ Benefits

| Aspect | Before | After |
|--------|--------|-------|
| **Add New Model** | Edit multiple files | Edit 1 place (registry) |
| **Code Duplication** | if-else for each model | Dynamic loop |
| **Maintainability** | Hard | Easy |
| **Scalability** | Limited | Unlimited |
| **Frontend** | Hardcoded UI | Dynamic rendering |

## 🎯 Summary

**To add a new model, you only need to:**
1. ✅ Add entry to `modelRegistry`
2. ✅ (Optional) Add fields to `NodeInfo`
3. ✅ (Optional) Add case in config handler
4. ✅ Implement on device side

**No need to:**
- ❌ Change endpoint logic
- ❌ Add if-else conditions
- ❌ Duplicate code
- ❌ Modify frontend (will auto-render)

---

**Version:** 2.0  
**Date:** 2025-12-17  
**Status:** ✅ Production Ready
