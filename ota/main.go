package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"
)

type NodeInfo struct {
	Status       string                 `json:"status,omitempty"`
	RamFreeBytes int64                  `json:"ram_free_bytes,omitempty"`
	SD_OK        *bool                  `json:"sd_ok,omitempty"`
	Ck           string                 `json:"ck,omitempty"`
	Area         string                 `json:"area,omitempty"`
	No           string                 `json:"no,omitempty"`
	MinT1        float64                `json:"min_t1,omitempty"`
	MaxT1        float64                `json:"max_t1,omitempty"`
	MinT2        float64                `json:"min_t2,omitempty"`
	MaxT2        float64                `json:"max_t2,omitempty"`
	MinT3        float64                `json:"min_t3,omitempty"`
	MaxT3        float64                `json:"max_t3,omitempty"`
	SHTSuhuMin   float64                `json:"sht_suhu_min,omitempty"`
	SHTSuhuMax   float64                `json:"sht_suhu_max,omitempty"`
	SHTHumMin    float64                `json:"sht_humidity_min,omitempty"`
	SHTHumMax    float64                `json:"sht_humidity_max,omitempty"`
	DoorDelay    uint32                 `json:"door_logic_delay,omitempty"`
	Reboot       int                    `json:"reboot,omitempty"`
	Interval     uint64                 `json:"interval,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	Updated      string                 `json:"updated,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Prefix       string                 `json:"node_prefix,omitempty"`
	Version      string                 `json:"version,omitempty"`
	AppMode      string                 `json:"app_mode,omitempty"`
	Trans        string                 `json:"trans,omitempty"`
	PassCode     string                 `json:"pass_code,omitempty"`
	Relay        bool                   `json:"relay"`
	NoT1         int                    `json:"no_t1"`
	NoT2         int                    `json:"no_t2"`
	NoT3         int                    `json:"no_t3"`
	NoSHT        int                    `json:"no_sht"`
	NoP1         int                    `json:"no_p1"`
	NoP2         int                    `json:"no_p2"`
	NoP3         int                    `json:"no_p3"`
	CurT1        float64                `json:"cur_t1"`
	CurT2        float64                `json:"cur_t2"`
	CurT3        float64                `json:"cur_t3"`
	CurP1        int                    `json:"cur_p1"`
	CurP2        int                    `json:"cur_p2"`
	CurP3        int                    `json:"cur_p3"`
	CurSHT_T     float64                `json:"cur_sht_t"`
	CurSHT_H     float64                `json:"cur_sht_h"`
	Logs         []string               `json:"logs,omitempty"`
	FullConfig   map[string]interface{} `json:"full_config,omitempty"`
}

type FileInfo struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	UploadTime time.Time `json:"upload_time"`
	Size       int64     `json:"size"`
}

// ModelConfig defines configuration fields for each model type
type ModelConfig struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Fields      []Field `json:"fields"`
	Command     string  `json:"command"` // MQTT command name
}

type Field struct {
	Name        string   `json:"name"`  // field name in JSON
	Label       string   `json:"label"` // display label
	Type        string   `json:"type"`  // "number", "text", "select"
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Step        *float64 `json:"step,omitempty"`
	Options     []string `json:"options,omitempty"` // for select type
}

var commonTempFields = []Field{
	{Name: "ck", Label: "Central Kitchen", Type: "number", Required: true},
	{Name: "area", Label: "Area ID", Type: "number", Required: true},
	{Name: "no", Label: "Box ID", Type: "number", Required: true},
	{Name: "no_t1", Label: "No T1", Type: "number", Required: false},
	{Name: "no_t2", Label: "No T2", Type: "number", Required: false},
	{Name: "no_t3", Label: "No T3", Type: "number", Required: false},
	{Name: "no_sht", Label: "No SHT", Type: "number", Required: false},
	{Name: "no_p1", Label: "No P1", Type: "number", Required: false},
	{Name: "no_p2", Label: "No P2", Type: "number", Required: false},
	{Name: "no_p3", Label: "No P3", Type: "number", Required: false},
	{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "TEMP"},
	{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
	{Name: "door_logic_delay", Label: "Door Alarm Delay (ms)", Type: "number", Required: false},
}

var commonMdcwFields = []Field{
	{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "MDCW"},
	{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
}

var modelRegistry = map[string]ModelConfig{
	"TEMP": {
		Name: "TEMP", DisplayName: "Temperature Sensor (Universal)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t3", Label: "Min T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t3", Label: "Max T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|1": {
		Name: "M1", DisplayName: "TEMP-M1 (1 DS)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|2": {
		Name: "M2", DisplayName: "TEMP-M2 (Modbus + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|3": {
		Name: "M3", DisplayName: "TEMP-M3 (2 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|4": {
		Name: "M4", DisplayName: "TEMP-M4 (3 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t3", Label: "Min T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t3", Label: "Max T3", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|5": {
		Name: "M5", DisplayName: "TEMP-M5 (1 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|6": {
		Name: "M6", DisplayName: "TEMP-M6 (1 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|7": {
		Name: "M7", DisplayName: "TEMP-M7 (1 Prox)", Command: "set_config",
		Fields: commonTempFields,
	},
	"TEMP|8": {
		Name: "M8", DisplayName: "TEMP-M8 (Modbus Only)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|9": {
		Name: "M9", DisplayName: "TEMP-M9 (1 DS Only)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|10": {
		Name: "M10", DisplayName: "TEMP-M10 (2 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|11": {
		Name: "M11", DisplayName: "TEMP-M11 (2 Prox)", Command: "set_config",
		Fields: commonTempFields,
	},
	"MDCW": {
		Name:        "MDCW",
		DisplayName: "MDCW Weighing",
		Command:     "set_config",
		Fields:      commonMdcwFields,
	},
	"V1": {
		Name:        "V1",
		DisplayName: "MDCW V1 (Standard)",
		Command:     "set_config",
		Fields:      commonMdcwFields,
	},
	"V2": {
		Name:        "V2",
		DisplayName: "MDCW V2 (Prox)",
		Command:     "set_config",
		Fields: append(commonMdcwFields, []Field{
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc2", Label: "Prox 3 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"TROLI": {
		Name:        "TROLI",
		DisplayName: "Troli Scanner (Mode A/B)",
		Command:     "set_config",
		Fields: []Field{
			{Name: "app_mode", Label: "App Mode", Type: "select", Required: true, Options: []string{"A", "B"}},
			{Name: "trans", Label: "Trans (IN/OUT) - Mode A Only", Type: "select", Required: false, Options: []string{"IN", "OUT"}},
			{Name: "pass_code", Label: "Target Product Code - Mode B", Type: "text", Required: false, Placeholder: "e.g., 100209"},
		},
	},
	"GENERIC": {
		Name: "GENERIC", DisplayName: "Unregistered Device", Command: "set_config",
		Fields: []Field{
			{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "NODE-"},
		},
	},
}

// Helper function for float pointer
func floatPtr(f float64) *float64 {
	return &f
}

var (
	mqttClient mqtt.Client
	nodeMutex  sync.RWMutex
	nodeStatus = make(map[string]*NodeInfo)
	fileMutex  sync.RWMutex
	fileInfos  = make(map[string]FileInfo)

	// Auth
	store            *session.Store
	telegramBotToken string
	telegramChatID   string

	// Webhook Key (Optional security)
	webhookToken string

	// Alarm Tracking
	lastAlarmState = make(map[string]bool)
	alarmMutex     sync.Mutex
)

type EMQXWebhook struct {
	Event    string `json:"event"`
	ClientID string `json:"clientid"`
	Reason   string `json:"reason,omitempty"`
}

var macRegex = regexp.MustCompile(`[0-9a-fA-F]{12}`)

// env helper
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	// ensure upload dir
	targetDir := filepath.Join("static", "uploads")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Fatalf("failed to create upload directory: %v", err)
	}

	// Clean registry and sync with disk on startup
	loadInitialFiles(targetDir)

	// --- Auth Config ---
	telegramBotToken = getEnv("TELE_BOT_OTA", "")
	telegramChatID = getEnv("TELEGRAM_CHAT_ID", "")
	webhookToken = getEnv("WEBHOOK_TOKEN", "")
	if telegramBotToken == "" || telegramChatID == "" {
		log.Println("Peringatan: TELE_BOT_OTA atau TELEGRAM_CHAT_ID tidak diatur. Fitur login tidak akan berfungsi.")
	}

	store = session.New(session.Config{
		Storage:        memory.New(),
		Expiration:     24 * time.Hour,
		KeyLookup:      "cookie:session_id",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		KeyGenerator: func() string {
			b := make([]byte, 16)
			rand.Read(b)
			return hex.EncodeToString(b)
		},
	})

	// Load existing files on startup
	loadInitialFiles("static/uploads")
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024, // 100MB Limit
	})

	// Serve raw download files
	app.Static("/files", "./static/uploads")

	// --- Public Routes ---
	app.Post("/api/webhook/emqx", handleEMQXWebhook)

	// --- Protected Routes (auth disabled) ---
	protected := app.Group("/")
	protected.Use(requireAuth)

	protected.Post("/logout", handleLogout)

	api := protected.Group("/api")

	// API: nodes snapshot
	api.Get("/nodes", func(c *fiber.Ctx) error {
		nodeMutex.RLock()
		defer nodeMutex.RUnlock()

		// Deduplication logic: only show the latest entry for each MAC address.
		// Assumes MAC is the last part of the node ID.
		latestNodes := make(map[string]*NodeInfo)
		macToNodeID := make(map[string]string)

		// Step 1: Find the latest node ID for each MAC address
		for id, info := range nodeStatus {
			matches := macRegex.FindAllString(id, -1)
			mac := id
			if len(matches) > 0 {
				mac = matches[len(matches)-1]
			}

			existingNodeID, found := macToNodeID[mac]
			if !found {
				macToNodeID[mac] = id
			} else if info.Updated > nodeStatus[existingNodeID].Updated {
				// This node is newer than the one we previously recorded for this MAC.
				// Update our record to point to this newer node ID.
				macToNodeID[mac] = id
			}
		}

		// Step 2: Build the final list of nodes from the winners identified in Step 1.
		for _, latestID := range macToNodeID {
			latestNodes[latestID] = nodeStatus[latestID]
		}

		// After deduplication, check for staleness
		now := time.Now()
		for _, info := range latestNodes {
			if info.Updated != "" {
				// FIX: Gunakan time.Local agar parsing sesuai dengan timezone server
				updatedTime, err := time.ParseInLocation("2006-01-02 15:04:05", info.Updated, time.Local)
				if err == nil {
					// Fallback: Jika tidak ada update selama 45 detik (Heartbeat alat 30s), tandai offline
					if info.Status != "offline" && now.Sub(updatedTime) > 45*time.Second {
						info.Status = "offline"
					}
				}
			} else {
				// No update record yet? Default to offline
				info.Status = "offline"
			}
		}

		return c.JSON(latestNodes)
	})

	// API: get model configurations
	api.Get("/models", func(c *fiber.Ctx) error {
		return c.JSON(modelRegistry)
	})

	// API: get specific model config
	api.Get("/models/:name", func(c *fiber.Ctx) error {
		modelName := c.Params("name")
		if config, exists := modelRegistry[modelName]; exists {
			return c.JSON(config)
		}

		// Prefix fallback
		modelPrefix := getEnv("MODEL_PREFIX", "TEMP|")
		if strings.HasPrefix(modelName, modelPrefix) {
			if config, exists := modelRegistry["TEMP"]; exists {
				return c.JSON(config)
			}
		}

		// Final fallback to GENERIC
		if config, exists := modelRegistry["GENERIC"]; exists {
			return c.JSON(config)
		}

		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	})

	// API: get fresh config from node (request it)
	api.Get("/nodes/:id/config", func(c *fiber.Ctx) error {
		id := c.Params("id")

		// Send get_config command
		payload := map[string]interface{}{"cmd": "get_config"}
		b, _ := json.Marshal(payload)
		mqttClient.Publish(fmt.Sprintf("nodes/%s/command", id), 0, false, b)

		// Return the latest FullConfig we have.
		nodeMutex.RLock()
		defer nodeMutex.RUnlock()
		if info, ok := nodeStatus[id]; ok {
			return c.JSON(info.FullConfig)
		}
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	})

	// DELETE node endpoint
	api.Delete("/nodes/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		nodeMutex.Lock()
		defer nodeMutex.Unlock()

		// Find the MAC of the node to be deleted
		matches := macRegex.FindAllString(id, -1)
		if len(matches) == 0 {
			// If no MAC found, just delete the specific node
			if _, ok := nodeStatus[id]; ok {
				delete(nodeStatus, id)
				// Clear its retained messages
				mqttClient.Publish(fmt.Sprintf("nodes/%s/status", id), 0, true, []byte{})
				mqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", id), 0, true, []byte{})
				return c.JSON(fiber.Map{"status": "deleted", "node": id})
			}
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "node not found"})
		}
		targetMac := matches[len(matches)-1]

		// Iterate over all nodes and delete any that match the target MAC
		nodesToDelete := []string{}
		deletedCount := 0
		for nodeID := range nodeStatus {
			nodeMacMatches := macRegex.FindAllString(nodeID, -1)
			if len(nodeMacMatches) > 0 && nodeMacMatches[len(nodeMacMatches)-1] == targetMac {
				nodesToDelete = append(nodesToDelete, nodeID)
			}
		}

		for _, nodeID := range nodesToDelete {
			delete(nodeStatus, nodeID)
			mqttClient.Publish(fmt.Sprintf("nodes/%s/status", nodeID), 0, true, []byte{})
			mqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", nodeID), 0, true, []byte{})
			deletedCount++
		}

		if deletedCount > 0 {
			return c.JSON(fiber.Map{"status": "deleted", "mac": targetMac, "count": deletedCount})
		}

		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no nodes found for the given ID or MAC"})
	})

	// API: files list
	api.Get("/files", func(c *fiber.Ctx) error {
		fileMutex.Lock()
		defer fileMutex.Unlock()

		targetDir := filepath.Join("static", "uploads")
		// 1. Get current files on disk
		entries, err := os.ReadDir(targetDir)
		if err == nil {
			diskFiles := make(map[string]bool)
			for _, entry := range entries {
				if !entry.IsDir() {
					diskFiles[entry.Name()] = true
				}
			}

			// 2. Clean registry: remove items from memory if they are no longer on disk
			tempMap := make(map[string]FileInfo)
			for name, info := range fileInfos {
				if diskFiles[name] {
					tempMap[name] = info
				} else {
					log.Printf("[SYNC] Removing stale entry: %s", name)
				}
			}
			fileInfos = tempMap

			// 3. Add to registry: if on disk but not in map (e.g. manual upload)
			for name := range diskFiles {
				if _, exists := fileInfos[name]; !exists {
					info, err := os.Stat(filepath.Join(targetDir, name))
					if err == nil {
						log.Printf("[SYNC] Adding missing disk file: %s", name)
						fileInfos[name] = FileInfo{
							Name:       name,
							URL:        "/files/" + name,
							UploadTime: info.ModTime(),
							Size:       info.Size(),
						}
					}
				}
			}
		} else {
			log.Printf("[SYNC] Warning: could not read upload directory: %v", err)
		}

		// 4. Return sorted/current list from memory
		files := make([]FileInfo, 0, len(fileInfos))
		for _, f := range fileInfos {
			files = append(files, f)
		}
		return c.JSON(files)
	})

	// DELETE file endpoint
	api.Delete("/files/*", func(c *fiber.Ctx) error {
		name := c.Params("*")
		if name == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename required"})
		}
		// security: prevent path traversal and ensure we only touch static/uploads
		clean := filepath.Base(name)
		path := filepath.Join("static", "uploads", clean)

		fileMutex.Lock()
		defer fileMutex.Unlock()

		if _, err := os.Stat(path); os.IsNotExist(err) {
			// If file is missing from disk, still remove it from internal map for sync
			delete(fileInfos, clean)
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found on disk, registry cleaned"})
		}
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to delete file %s: %v", path, err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed delete"})
		}
		delete(fileInfos, clean)
		return c.JSON(fiber.Map{"status": "deleted", "name": clean})
	})

	// RENAME file endpoint
	api.Post("/files/:name/rename", func(c *fiber.Ctx) error {
		name := c.Params("name")
		type RenameRequest struct {
			NewName string `json:"new_name"`
		}
		var req RenameRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		// Sanitize new file name to prevent path traversal or invalid names
		cleanNewName := filepath.Base(req.NewName)
		if cleanNewName == "" || cleanNewName == "." || cleanNewName == ".." {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid new name"})
		}

		fileMutex.Lock()
		defer fileMutex.Unlock()

		if _, ok := fileInfos[name]; !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}

		oldPath := filepath.Join("static", "uploads", name)
		newPath := filepath.Join("static", "uploads", cleanNewName)

		if err := os.Rename(oldPath, newPath); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rename file"})
		}

		fileInfo := fileInfos[name]
		delete(fileInfos, name)
		fileInfo.Name = cleanNewName
		fileInfo.URL = "/files/" + cleanNewName
		fileInfos[cleanNewName] = fileInfo

		return c.JSON(fileInfo)
	})

	// Upload OTA (form multipart)
	// Upload OTA (form multipart) - Multi-file support
	protected.Post("/upload", func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("Upload failed: %v", err)
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "multipart form required"})
		}

		files := form.File["file"]
		if len(files) == 0 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "no files provided"})
		}

		var uploadedFiles []string
		fileMutex.Lock()
		defer fileMutex.Unlock()

		for _, f := range files {
			// Security: Always use Base name to prevent path traversal and ensure consistency
			baseName := filepath.Base(f.Filename)
			dst := filepath.Join("static", "uploads", baseName)

			if err := c.SaveFile(f, dst); err != nil {
				log.Printf("Failed to save file %s: %v", dst, err)
				continue // Skip failed files but continue with others
			}

			fileInfos[baseName] = FileInfo{
				Name:       baseName,
				URL:        "/files/" + baseName,
				UploadTime: time.Now(),
				Size:       f.Size,
			}
			uploadedFiles = append(uploadedFiles, baseName)
			log.Printf("Successfully uploaded: %s as %s (%d bytes)", f.Filename, baseName, f.Size)
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"count":  len(uploadedFiles),
			"files":  uploadedFiles,
		})
	})

	// Config -> publish to nodes/{id}/command (Dynamic config)
	protected.Post("/config", func(c *fiber.Ctx) error {
		var req map[string]interface{}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}

		nodeID, ok := req["node"].(string)
		if !ok || nodeID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "node ID required"})
		}

		// Prepare dynamic payload
		payload := make(map[string]interface{})
		for k, v := range req {
			if k == "node" {
				continue
			}
			payload[k] = v
		}

		if _, exists := payload["cmd"]; !exists {
			payload["cmd"] = "set_config"
		}

		// Update nodeStatus memory map dynamically
		nodeMutex.Lock()
		if info, ok := nodeStatus[nodeID]; ok {
			if info.FullConfig == nil {
				info.FullConfig = make(map[string]interface{})
			}
			for k, v := range payload {
				if k == "cmd" {
					continue
				}
				info.FullConfig[k] = v
			}
		}
		nodeMutex.Unlock()

		// Marshal and publish
		b, err := json.Marshal(payload)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payload"})
		}

		topic := fmt.Sprintf("nodes/%s/command", nodeID)
		token := mqttClient.Publish(topic, 0, false, b)
		token.Wait()

		return c.JSON(fiber.Map{
			"status": "ok",
			"topic":  topic,
			"cmd":    payload["cmd"],
		})
	})

	// OTA trigger
	protected.Post("/ota", func(c *fiber.Ctx) error {
		type O struct {
			Node string `json:"node"`
			URL  string `json:"url"`
		}
		var o O
		if err := c.BodyParser(&o); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		payload := map[string]interface{}{"cmd": "ota", "url": o.URL}
		b, err := json.Marshal(payload)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payload"})
		}
		topic := fmt.Sprintf("nodes/%s/command", o.Node)
		token := mqttClient.Publish(topic, 0, false, b)
		token.Wait()
		return c.JSON(fiber.Map{"status": "OTA triggered", "topic": topic})
	})

	// Reboot trigger
	protected.Post("/reboot", func(c *fiber.Ctx) error {
		type R struct {
			Node string `json:"node"`
		}
		var r R
		if err := c.BodyParser(&r); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		payload := map[string]interface{}{"cmd": "reboot"}
		b, err := json.Marshal(payload)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payload"})
		}
		topic := fmt.Sprintf("nodes/%s/command", r.Node)
		token := mqttClient.Publish(topic, 0, false, b)
		token.Wait()
		return c.JSON(fiber.Map{"status": "Reboot triggered", "topic": topic})
	})

	// logs endpoint (last 3 lines)
	protected.Get("/logs/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		nodeMutex.RLock()
		defer nodeMutex.RUnlock()
		if info, ok := nodeStatus[id]; ok {
			return c.JSON(fiber.Map{"node": id, "logs": info.Logs})
		}
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	})

	// Forwarder proxy endpoint
	protected.Get("/forwarder/status", func(c *fiber.Ctx) error {
		forwarderURL := getEnv("FORWARDER_URL", "http://forwarder:8888/forwarder/status")
		resp, err := http.Get(forwarderURL)
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Set("Content-Type", "application/json")
		return c.Send(body)
	})

	// Monitor proxy endpoint
	protected.Get("/monitor/status", func(c *fiber.Ctx) error {
		monitorURL := getEnv("MONITOR_URL", "http://monitor:9090/status")
		resp, err := http.Get(monitorURL)
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Monitor service tidak tersedia"})
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Set("Content-Type", "application/json")
		return c.Send(body)
	})

	// Pipeline CRUD proxies
	protected.Get("/api/pipelines", func(c *fiber.Ctx) error {
		forwarderBase := getEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
		resp, err := http.Get(forwarderBase + "/pipelines")
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Set("Content-Type", "application/json")
		return c.Send(body)
	})

	protected.Post("/api/pipelines", func(c *fiber.Ctx) error {
		forwarderBase := getEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
		resp, err := http.Post(forwarderBase+"/pipelines", "application/json", strings.NewReader(string(c.Body())))
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Status(resp.StatusCode).Set("Content-Type", "application/json")
		return c.Send(body)
	})

	protected.Delete("/api/pipelines/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		forwarderBase := getEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
		req, _ := http.NewRequest("DELETE", forwarderBase+"/pipelines/"+id, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
		}
		defer resp.Body.Close()
		return c.SendStatus(resp.StatusCode)
	})

	// Start MQTT connection (non-blocking)
	go initMQTT()

	// --- Serve Svelte Dashboard (SPA) ---
	// The built Svelte app lives in ./web (copied during Docker build from dashboard/build/)
	app.Static("/", "./web")
	// SPA fallback – unmatched non-API paths return index.html
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})

	// Start background cleanup for very old offline nodes (e.g., every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			nodeMutex.Lock()
			now := time.Now()
			for id, info := range nodeStatus {
				if info.Updated != "" {
					updatedTime, err := time.Parse("2006-01-02 15:04:05", info.Updated)
					if err == nil && now.Sub(updatedTime) > 7*24*time.Hour {
						log.Printf("Cleaning up old node: %s", id)
						delete(nodeStatus, id)
					}
				}
			}
			nodeMutex.Unlock()
		}
	}()

	// Listen
	log.Fatal(app.Listen("0.0.0.0:9999"))
}

func requireAuth(c *fiber.Ctx) error {
	// DEBUG: Auth Tele temporarily disabled
	return c.Next()

	/*
		sess, err := store.Get(c)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString("Session error")
		}

		if sess.Get("authenticated") != true {
			return c.Redirect("/login")
		}

		return c.Next()
	*/
}

func handleShowLogin(c *fiber.Ctx) error {
	return c.Render("login", nil)
}

func handleLogin(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	submittedCode := strings.ToLower(c.FormValue("code"))
	if submittedCode == "" {
		return c.Render("login", fiber.Map{"error": "Kode tidak boleh kosong."})
	}

	// Ambil kode dari sesi
	authCodeVal := sess.Get("auth_code")
	authExpiresVal := sess.Get("auth_expires")

	if authCodeVal == nil || authExpiresVal == nil {
		return c.Render("login", fiber.Map{"error": "Sesi tidak ditemukan. Silakan minta kode baru."})
	}

	authCode, ok1 := authCodeVal.(string)

	// Robust expiry check (handle int, int64, float64 types)
	var expiryUnix int64
	var ok2 bool
	switch v := authExpiresVal.(type) {
	case int64:
		expiryUnix = v
		ok2 = true
	case int:
		expiryUnix = int64(v)
		ok2 = true
	case float64:
		expiryUnix = int64(v)
		ok2 = true
	}

	if !ok1 || !ok2 {
		return c.Render("login", fiber.Map{"error": "Data sesi korup. Silakan minta kode baru."})
	}

	expiryTime := time.Unix(expiryUnix, 0)

	if authCode != submittedCode || time.Now().After(expiryTime) {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kadaluarsa."})
	}

	// Kode valid, hapus dari sesi dan set status login
	sess.Delete("auth_code")
	sess.Delete("auth_expires")
	sess.Set("authenticated", true)
	if err := sess.Save(); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Gagal menyimpan sesi")
	}

	return c.Redirect("/")
}

func handleLogout(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}
	sess.Destroy()
	return c.Redirect("/login")
}

func handleRequestCode(c *fiber.Ctx) error {
	if telegramBotToken == "" || telegramChatID == "" {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Layanan Telegram tidak dikonfigurasi di server.",
		})
	}

	sess, err := store.Get(c)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Gagal membuat sesi."})
	}

	// Buat kode acak 6 digit
	b := make([]byte, 3)
	rand.Read(b)
	code := hex.EncodeToString(b)

	// Simpan kode dan expiry sebagai Unix timestamp (int64)
	sess.Set("auth_code", code)
	sess.Set("auth_expires", time.Now().Add(5*time.Minute).Unix())

	if err := sess.Save(); err != nil {
		log.Printf("Error saving session: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": fmt.Sprintf("Gagal menyimpan sesi: %v", err)})
	}

	log.Printf("Code generated and saved: %s", code)

	// Kirim kode ke Telegram dalam gaya JSON
	message := fmt.Sprintf("```json\n{\n  \"event\": \"AUTH_CODE_GENERATED\",\n  \"service\": \"OTA_CORE\",\n  \"auth_code\": \"%s\",\n  \"expires\": \"5m\",\n  \"status\": \"pending\"\n}\n```", code)
	go sendTelegramMessage(message)

	return c.JSON(fiber.Map{"status": "ok"})
}

func sendTelegramMessage(message string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)

	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {telegramChatID},
		"text":       {message},
		"parse_mode": {"Markdown"},
	})

	if err != nil {
		log.Printf("Error sending Telegram message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send Telegram message, status: %s, response: %s", resp.Status, string(body))
	}
}

func loadInitialFiles(dir string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("could not read upload directory %s: %v", dir, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				name := info.Name()
				fileInfos[name] = FileInfo{
					Name:       name,
					URL:        "/files/" + name,
					UploadTime: info.ModTime(),
					Size:       info.Size(),
				}
			}
		}
	}
}

func initMQTT() {
	// broker and creds
	brokerHost := getEnv("MQTT_BROKER", "")
	if brokerHost == "" {
		log.Fatal("MQTT_BROKER environment variable is required")
	}
	mqttUser := getEnv("MQTT_USER", "apps")
	mqttPass := getEnv("MQTT_PASS", "apps")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerHost)
	opts.SetClientID(fmt.Sprintf("web-%d", time.Now().Unix()))
	if mqttUser != "" {
		opts.SetUsername(mqttUser)
	}
	if mqttPass != "" {
		opts.SetPassword(mqttPass)
	}
	opts.AutoReconnect = true
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT connected to", brokerHost)
		// subscribe status/topic patterns
		if token := c.Subscribe("nodes/+/status", 0, mqttHandler); token.Wait() && token.Error() != nil {
			log.Println("subscribe status err:", token.Error())
		}
		if token := c.Subscribe("nodes/+/monitor", 0, mqttHandler); token.Wait() && token.Error() != nil {
			log.Println("subscribe nodes/+/monitor err:", token.Error())
		}
		// support firmware publishing to "<node>/monitor" or "nodes/<node>/monitor"
		if token := c.Subscribe("+/monitor", 0, mqttHandler); token.Wait() && token.Error() != nil {
			log.Println("subscribe +/monitor err:", token.Error())
		}
		// subscribe logs
		if token := c.Subscribe("nodes/+/log", 0, mqttHandler); token.Wait() && token.Error() != nil {
			log.Println("subscribe nodes/+/log err:", token.Error())
		}
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Println("MQTT lost:", err)
	}

	mqttClient = mqtt.NewClient(opts)
	for {
		if token := mqttClient.Connect(); token.Wait() && token.Error() == nil {
			break
		} else {
			log.Println("waiting for mqtt broker, retry in 2s...")
			time.Sleep(2 * time.Second)
		}
	}
}

// mqttHandler parses status, monitor, log messages
func mqttHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	parts := strings.Split(topic, "/")
	if len(parts) < 2 {
		return
	}

	var nodeID, sub string
	if parts[0] == "nodes" && len(parts) >= 3 {
		nodeID = parts[1]
		sub = parts[2]
	} else if len(parts) >= 2 {
		nodeID = parts[0]
		sub = parts[1]
	} else {
		return
	}

	raw := msg.Payload()
	now := time.Now().Format("2006-01-02 15:04:05")

	nodeMutex.Lock()
	defer nodeMutex.Unlock()

	// --- Smart Node Migration Logic ---
	// If a new nodeID appears for an existing MAC, migrate config and delete the old one.
	if _, ok := nodeStatus[nodeID]; !ok {
		newMacMatches := macRegex.FindAllString(nodeID, -1)
		if len(newMacMatches) > 0 {
			newMac := newMacMatches[len(newMacMatches)-1]
			for oldID, oldInfo := range nodeStatus {
				if oldID == nodeID {
					continue
				}
				oldMacMatches := macRegex.FindAllString(oldID, -1)
				if len(oldMacMatches) > 0 && oldMacMatches[len(oldMacMatches)-1] == newMac {
					log.Printf("MAC match! Migrating '%s' -> '%s'", oldID, nodeID)
					// Copy all fields to new node ID entry
					newNodeInfo := &NodeInfo{
						Ck:           oldInfo.Ck,
						Area:         oldInfo.Area,
						No:           oldInfo.No,
						MinT1:        oldInfo.MinT1,
						MaxT1:        oldInfo.MaxT1,
						MinT2:        oldInfo.MinT2,
						MaxT2:        oldInfo.MaxT2,
						MinT3:        oldInfo.MinT3,
						MaxT3:        oldInfo.MaxT3,
						SHTSuhuMin:   oldInfo.SHTSuhuMin,
						SHTSuhuMax:   oldInfo.SHTSuhuMax,
						SHTHumMin:    oldInfo.SHTHumMin,
						SHTHumMax:    oldInfo.SHTHumMax,
						Interval:     oldInfo.Interval,
						Prefix:       oldInfo.Prefix,
						Model:        oldInfo.Model,
						Version:      oldInfo.Version,
						AppMode:      oldInfo.AppMode,
						Trans:        oldInfo.Trans,
						PassCode:     oldInfo.PassCode,
						IP:           oldInfo.IP,
						RamFreeBytes: oldInfo.RamFreeBytes,
						SD_OK:        oldInfo.SD_OK,
						Logs:         oldInfo.Logs,
						Status:       "online",
						Updated:      now,
					}
					nodeStatus[nodeID] = newNodeInfo
					delete(nodeStatus, oldID)
					break
				}
			}
		}
	}

	if _, ok := nodeStatus[nodeID]; !ok {
		nodeStatus[nodeID] = &NodeInfo{}
	}
	info := nodeStatus[nodeID]

	switch sub {
	case "status":
		var tmp interface{}
		if err := json.Unmarshal(raw, &tmp); err == nil {
			if m, ok := tmp.(map[string]interface{}); ok {
				// Simpan semua data asli ke FullConfig (Fully Dynamic)
				info.FullConfig = m

				// Auto-map matching fields ke struct NodeInfo menggunakan json tags
				// Ini menghemat ratusan baris kode pemetaan manual
				if err := json.Unmarshal(raw, info); err != nil {
					log.Printf("[MQTT] Error auto-mapping node info: %v", err)
				}

				// Handle mapping khusus untuk field yang mungkin punya nama berbeda di JSON lama
				if v, ok := m["active_model"]; ok {
					info.Model = fmt.Sprintf("%v", v)
				}

				// Handle nested "conf" atau "config" jika ada (backward compatibility)
				for _, key := range []string{"conf", "config"} {
					if cfg, ex := m[key]; ex {
						if cm, ok := cfg.(map[string]interface{}); ok {
							// Merge nested config ke FullConfig agar muncul di UI
							for k, v := range cm {
								info.FullConfig[k] = v
							}
							// Re-unmarshal nested content untuk mengisi field struct yang cocok
							cfgRaw, _ := json.Marshal(cm)
							json.Unmarshal(cfgRaw, info)
						}
					}
				}
			} else {
				info.Status = fmt.Sprintf("%v", tmp)
			}
			info.Updated = now
		}
	case "monitor":
		// User: "serial monitor dari /monitor"
		// Append monitor payload to logs as well
		monStr := string(raw)
		info.Logs = append(info.Logs, "[MON] "+monStr)
		if len(info.Logs) > 10 { // Allow more logs for serial monitor
			info.Logs = info.Logs[len(info.Logs)-10:]
		}

		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err == nil {
			info.FullConfig = m
			if v, ok := m["ram"]; ok {
				if val, ok := v.(float64); ok {
					info.RamFreeBytes = int64(val)
				}
			}
			if v, ok := m["ram_free"]; ok {
				if val, ok := v.(float64); ok {
					info.RamFreeBytes = int64(val)
				}
			}
			if v, ok := m["ram_free_bytes"]; ok {
				if val, ok := v.(float64); ok {
					info.RamFreeBytes = int64(val)
				}
			}
			if v, ok := m["ip"]; ok {
				info.IP = fmt.Sprintf("%v", v)
			}
			if v, ok := m["ck"]; ok {
				info.Ck = fmt.Sprintf("%v", v)
			}
			if v, ok := m["area"]; ok {
				info.Area = fmt.Sprintf("%v", v)
			}
			if v, ok := m["no"]; ok {
				info.No = fmt.Sprintf("%v", v)
			}
			if v, ok := m["no_t1"]; ok {
				if f, ok := v.(float64); ok {
					info.NoT1 = int(f)
				}
			}
			if v, ok := m["no_t2"]; ok {
				if f, ok := v.(float64); ok {
					info.NoT2 = int(f)
				}
			}
			if v, ok := m["no_t3"]; ok {
				if f, ok := v.(float64); ok {
					info.NoT3 = int(f)
				}
			}
			if v, ok := m["no_sht"]; ok {
				if f, ok := v.(float64); ok {
					info.NoSHT = int(f)
				}
			}
			if v, ok := m["no_p1"]; ok {
				if f, ok := v.(float64); ok {
					info.NoP1 = int(f)
				}
			}
			if v, ok := m["no_p2"]; ok {
				if f, ok := v.(float64); ok {
					info.NoP2 = int(f)
				}
			}
			if v, ok := m["no_p3"]; ok {
				if f, ok := v.(float64); ok {
					info.NoP3 = int(f)
				}
			}
			if v, ok := m["sd_ok"]; ok {
				if b, ok := v.(bool); ok {
					info.SD_OK = &b
				}
			}
			if v, ok := m["model"]; ok {
				info.Model = fmt.Sprintf("%v", v)
			}
			if v, ok := m["version"]; ok {
				info.Version = fmt.Sprintf("%v", v)
			}
			if v, ok := m["node_prefix"]; ok {
				info.Prefix = fmt.Sprintf("%v", v)
			}
			if data, ok := m["data"]; ok {
				if dm, ok := data.(map[string]interface{}); ok {
					if t1, ok := dm["t1"].(float64); ok {
						info.CurT1 = t1
					}
					if t2, ok := dm["t2"].(float64); ok {
						info.CurT2 = t2
					}
					if t3, ok := dm["t3"].(float64); ok {
						info.CurT3 = t3
					}
					if p1, ok := dm["p1"].(float64); ok {
						info.CurP1 = int(p1)
					}
					if p2, ok := dm["p2"].(float64); ok {
						info.CurP2 = int(p2)
					}
					if p3, ok := dm["p3"].(float64); ok {
						info.CurP3 = int(p3)
					}
					if st, ok := dm["sht_t"].(float64); ok {
						info.CurSHT_T = st
					}
					if sh, ok := dm["sht_h"].(float64); ok {
						info.CurSHT_H = sh
					}
				}
			}
			if v, ok := m["relay"]; ok {
				relayOn := false
				switch val := v.(type) {
				case bool:
					relayOn = val
				case float64:
					relayOn = val > 0
				case string:
					relayOn = val == "ON" || val == "1" || val == "true"
				}

				if relayOn != info.Relay {
					info.Relay = relayOn
					alarmMutex.Lock()
					prev, exists := lastAlarmState[nodeID]
					if relayOn && (!exists || !prev) {
						lastAlarmState[nodeID] = true
						go sendRelayTelegram(nodeID, info, true)
					} else if !relayOn && exists && prev {
						lastAlarmState[nodeID] = false
						go sendRelayTelegram(nodeID, info, false)
					}
					alarmMutex.Unlock()
				}
			}
		}
		info.Updated = now

	case "log":
		line := string(raw)
		line = strings.TrimSpace(line)
		if line != "" {
			info.Logs = append(info.Logs, "[LOG] "+line)
			if len(info.Logs) > 10 {
				info.Logs = info.Logs[len(info.Logs)-10:]
			}
			info.Updated = now
		}
	default:
		// ignore
	}
	nodeStatus[nodeID] = info
}

func handleEMQXWebhook(c *fiber.Ctx) error {
	// Optional security check
	if webhookToken != "" && c.Get("X-Webhook-Token") != webhookToken {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var payload EMQXWebhook
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse body"})
	}

	nodeID := payload.ClientID
	if nodeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty clientid"})
	}

	// FILTER: Ignore technical client IDs unless they already exist as valid nodes
	isTechnical := strings.HasPrefix(nodeID, "web-") ||
		strings.HasPrefix(nodeID, "megdev-") ||
		strings.HasPrefix(nodeID, "servfi-") ||
		strings.HasPrefix(nodeID, "forming-") ||
		strings.HasPrefix(nodeID, "forwarder-")

	nodeMutex.Lock()
	defer nodeMutex.Unlock()

	exists := false
	if _, ok := nodeStatus[nodeID]; ok {
		exists = ok
	}

	if isTechnical && !exists {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "technical_client"})
	}

	if !exists {
		nodeStatus[nodeID] = &NodeInfo{}
	}
	info := nodeStatus[nodeID]

	now := time.Now().Format("2006-01-02 15:04:05")
	info.Updated = now

	switch payload.Event {
	case "client.connected":
		info.Status = "online"
		log.Printf("Webhook: Node %s connected", nodeID)
	case "client.disconnected":
		info.Status = "offline"
		log.Printf("Webhook: Node %s disconnected (reason: %s)", nodeID, payload.Reason)
	default:
		return c.JSON(fiber.Map{"status": "ignored", "event": payload.Event})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func sendRelayTelegram(nodeID string, info *NodeInfo, isActive bool) {
	statusStr := "🚨 *ALARM ACTIVE*"
	if !isActive {
		statusStr = "✅ *ALARM CLEARED*"
	}

	emoji := "🔋"
	if isActive {
		emoji = "⚠️"
	}

	message := fmt.Sprintf("%s\n", statusStr)
	message += fmt.Sprintf("*Device:* `%s` (%s)\n", nodeID, info.Model)
	message += fmt.Sprintf("*Site:* CK %s - Area %s (Node #%s)\n", info.Ck, info.Area, info.No)
	message += "----------------------------\n"

	// Sensor Readings
	message += "*TEMPERATURES:*\n"
	if info.CurT1 > -100 {
		message += fmt.Sprintf("• T1: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT1, info.MinT1, info.MaxT1)
	}
	if info.CurT2 > -100 {
		message += fmt.Sprintf("• T2: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT2, info.MinT2, info.MaxT2)
	}
	if info.CurT3 > -100 {
		message += fmt.Sprintf("• T3: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT3, info.MinT3, info.MaxT3)
	}

	if info.CurSHT_T > -40 {
		message += fmt.Sprintf("• SHT: `%.1f°C` / `%.1f%%`RH\n", info.CurSHT_T, info.CurSHT_H)
	}

	message += "\n*DOOR STATUS:*\n"
	message += fmt.Sprintf("• P1: %s\n", formatDoor(info.CurP1))
	message += fmt.Sprintf("• P2: %s\n", formatDoor(info.CurP2))
	message += fmt.Sprintf("• P3: %s\n", formatDoor(info.CurP3))

	message += "----------------------------\n"
	if isActive {
		message += fmt.Sprintf("%s *RELAY STATUS: ON*\n", emoji)
		message += "_Immediately check the storage area!_"
	} else {
		message += "🔋 *RELAY STATUS: OFF*\n"
		message += "_System normalized._"
	}

	sendTelegramMessage(message)
}

func formatDoor(val int) string {
	if val == 0 {
		return "🟢 CLOSED"
	}
	return "🔴 OPEN"
}
