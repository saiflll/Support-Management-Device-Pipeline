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
	"github.com/gofiber/template/html/v2"
)

type NodeInfo struct {
	Status       string   `json:"status,omitempty"`
	RamFreeBytes int64    `json:"ram_free_bytes,omitempty"`
	SD_OK        *bool    `json:"sd_ok,omitempty"` // Use pointer to distinguish between false and not set
	Ck           string   `json:"ck,omitempty"`
	Area         string   `json:"area,omitempty"`
	No           string   `json:"no,omitempty"`
	Min          float64  `json:"min,omitempty"`
	Max          float64  `json:"max,omitempty"`
	Min0         float64  `json:"min0,omitempty"`
	Max0         float64  `json:"max0,omitempty"`
	Min1         float64  `json:"min1,omitempty"`
	Max1         float64  `json:"max1,omitempty"`
	Min2         float64  `json:"min2,omitempty"`
	Max2         float64  `json:"max2,omitempty"`
	Min3         float64  `json:"min3,omitempty"`
	Max3         float64  `json:"max3,omitempty"`
	Min4         float64  `json:"min4,omitempty"`
	Max4         float64  `json:"max4,omitempty"`
	ProxNc0      int      `json:"prox_nc0"`
	ProxNc1      int      `json:"prox_nc1"`
	ProxNc2      int      `json:"prox_nc2"`
	Interval     uint64   `json:"interval,omitempty"`
	IP           string   `json:"ip,omitempty"`
	Updated      string   `json:"updated,omitempty"`
	Model        string   `json:"model,omitempty"`
	Prefix       string   `json:"prefix,omitempty"`
	Version      string   `json:"version,omitempty"`
	AppMode      string   `json:"app_mode,omitempty"`
	Trans        string   `json:"trans,omitempty"`
	PassCode     string   `json:"pass_code,omitempty"`
	Logs         []string `json:"logs,omitempty"` // last 3 log lines
}

type FileInfo struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	UploadTime time.Time `json:"upload_time"`
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

// Model registry - easy to add new models
var commonTempFields = []Field{
	{Name: "ck", Label: "Central Kitchen", Type: "number", Required: true},
	{Name: "area", Label: "Area ID", Type: "number", Required: true},
	{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
}

var modelRegistry = map[string]ModelConfig{
	"TEMP": {
		Name: "TEMP", DisplayName: "Temperature Sensor (Base)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"M1": {
		Name: "M1", DisplayName: "TEMP-M1 (1 DS)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"M2": {
		Name: "M2", DisplayName: "TEMP-M2 (Modbus + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M3": {
		Name: "M3", DisplayName: "TEMP-M3 (2 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min1", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max1", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M4": {
		Name: "M4", DisplayName: "TEMP-M4 (3 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min1", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max1", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min2", Label: "Min T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max2", Label: "Max T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M5": {
		Name: "M5", DisplayName: "TEMP-M5 (1 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M6": {
		Name: "M6", DisplayName: "TEMP-M6 (1 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M7": {
		Name: "M7", DisplayName: "TEMP-M7 (1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M8": {
		Name: "M8", DisplayName: "TEMP-M8 (Modbus Only)", Command: "set_config",
		Fields: commonTempFields,
	},
	"M9": {
		Name: "M9", DisplayName: "TEMP-M9 (1 DS Only)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"M10": {
		Name: "M10", DisplayName: "TEMP-M10 (2 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min0", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max0", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min1", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max1", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"M11": {
		Name: "M11", DisplayName: "TEMP-M11 (2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"MDCW": {
		Name:        "MDCW",
		DisplayName: "MDCW Weighing",
		Command:     "set_config",
		Fields: []Field{
			{Name: "prefix", Label: "Device Prefix", Type: "text", Required: true, Placeholder: "e.g., MDCW_01"},
			{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
		},
	},
	"V1": {
		Name:        "V1",
		DisplayName: "MDCW V1 (Standard)",
		Command:     "set_config",
		Fields: []Field{
			{Name: "prefix", Label: "Device Prefix", Type: "text", Required: true, Placeholder: "e.g., MDCW_01"},
			{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
		},
	},
	"V2": {
		Name:        "V2",
		DisplayName: "MDCW V2 (Prox)",
		Command:     "set_config",
		Fields: []Field{
			{Name: "prefix", Label: "Device Prefix", Type: "text", Required: true, Placeholder: "e.g., MDCW_01"},
			{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc2", Label: "Prox 3 (0:NO, 1:NC)", Type: "number", Required: false},
		},
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
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views:     engine,
		BodyLimit: 100 * 1024 * 1024, // 100MB Limit
	})

	// static assets & files
	app.Static("/static", "./static")
	app.Static("/files", "./static/uploads")

	// --- Public Routes ---
	app.Get("/login", handleShowLogin)
	app.Post("/login", handleLogin)
	app.Post("/request-code", handleRequestCode)
	app.Post("/api/webhook/emqx", handleEMQXWebhook)

	// --- Protected Routes ---
	// Grup ini memerlukan autentikasi
	protected := app.Group("/")
	protected.Use(requireAuth)

	protected.Get("/", func(c *fiber.Ctx) error {
		brokerHost := getEnv("MQTT_BROKER", "")
		serverName := getEnv("SERVER_NAME", "ren_itdt_west")
		return c.Render("index", fiber.Map{
			"broker":     brokerHost,
			"serverName": serverName,
		})
	})

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
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
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
		diskFiles := make(map[string]bool)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					diskFiles[entry.Name()] = true
				}
			}
		}

		// 2. Clean registry: remove if not on disk
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
					}
				}
			}
		}

		// 4. Return sorted/current list
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
	protected.Post("/upload", func(c *fiber.Ctx) error {
		f, err := c.FormFile("file")
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString("file required")
		}
		dst := filepath.Join("static", "uploads", filepath.Base(f.Filename))
		if err := c.SaveFile(f, dst); err != nil {
			return err
		}

		fileMutex.Lock()
		defer fileMutex.Unlock()
		fileInfos[f.Filename] = FileInfo{
			Name:       f.Filename,
			URL:        "/files/" + f.Filename,
			UploadTime: time.Now(),
		}

		return c.JSON(fiber.Map{"status": "ok", "filename": f.Filename})
	})

	// Config -> publish to nodes/{id}/command (Dynamic model-based)
	protected.Post("/config", func(c *fiber.Ctx) error {
		var req map[string]interface{}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}

		nodeID, ok := req["node"].(string)
		if !ok || nodeID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "node ID required"})
		}

		// Get model from nodeStatus
		nodeMutex.RLock()
		model := ""
		version := ""
		if info, ok := nodeStatus[nodeID]; ok {
			model = info.Model
			version = info.Version
		}
		nodeMutex.RUnlock()

		// Smart model detection: use variant from version if available
		activeModel := model
		if version != "" && strings.Contains(version, "-") {
			parts := strings.Split(version, "-")
			variant := parts[len(parts)-1]
			if _, exists := modelRegistry[variant]; exists {
				activeModel = variant
			}
		}

		// Default to TEMP if no model specified
		if activeModel == "" {
			activeModel = "TEMP"
		}

		// Get model config from registry
		modelConfig, exists := modelRegistry[activeModel]
		if !exists {
			// Second fallback to base model
			modelConfig, exists = modelRegistry[model]
			if !exists {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("unknown model: %s", activeModel),
				})
			}
		}

		// Build payload dynamically based on model fields
		payload := map[string]interface{}{
			"cmd": modelConfig.Command,
		}

		// Extract field values from request
		for _, field := range modelConfig.Fields {
			if value, ok := req[field.Name]; ok {
				payload[field.Name] = value

				// Update nodeStatus with new values
				nodeMutex.Lock()
				if info, ok := nodeStatus[nodeID]; ok {
					switch field.Name {
					case "ck":
						info.Ck = fmt.Sprint(value)
					case "area":
						info.Area = fmt.Sprint(value)
					case "no":
						info.No = fmt.Sprint(value)
					case "prefix":
						info.Prefix = fmt.Sprint(value)
					case "interval":
						if f, ok := value.(float64); ok {
							info.Interval = uint64(f)
						}
					case "prox_nc0":
						if f, ok := value.(float64); ok {
							info.ProxNc0 = int(f)
						}
					case "prox_nc1":
						if f, ok := value.(float64); ok {
							info.ProxNc1 = int(f)
						}
					case "prox_nc2":
						if f, ok := value.(float64); ok {
							info.ProxNc2 = int(f)
						}
					case "app_mode":
						info.AppMode = fmt.Sprint(value)
					case "trans":
						info.Trans = fmt.Sprint(value)
					case "pass_code":
						info.PassCode = fmt.Sprint(value)
					// --- Fix: Added missing threshold fields ---
					case "min":
						if f, ok := value.(float64); ok {
							info.Min = f
						}
					case "max":
						if f, ok := value.(float64); ok {
							info.Max = f
						}
					case "min0":
						if f, ok := value.(float64); ok {
							info.Min0 = f
						}
					case "max0":
						if f, ok := value.(float64); ok {
							info.Max0 = f
						}
					case "min1":
						if f, ok := value.(float64); ok {
							info.Min1 = f
						}
					case "max1":
						if f, ok := value.(float64); ok {
							info.Max1 = f
						}
					case "min2":
						if f, ok := value.(float64); ok {
							info.Min2 = f
						}
					case "max2":
						if f, ok := value.(float64); ok {
							info.Max2 = f
						}
					case "min3":
						if f, ok := value.(float64); ok {
							info.Min3 = f
						}
					case "max3":
						if f, ok := value.(float64); ok {
							info.Max3 = f
						}
					case "min4":
						if f, ok := value.(float64); ok {
							info.Min4 = f
						}
					case "max4":
						if f, ok := value.(float64); ok {
							info.Max4 = f
						}
					}
				}
				nodeMutex.Unlock()
			} else if field.Required {
				return c.Status(400).JSON(fiber.Map{
					"error": fmt.Sprintf("required field missing: %s", field.Name),
				})
			}
		}

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
			"model":  model,
			"cmd":    modelConfig.Command,
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
		forwarderURL := getEnv("FORWARDER_URL", "http://backend:8000/forwarder/status")

		resp, err := http.Get(forwarderURL)
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Forwarder service tidak tersedia",
			})
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal membaca response dari forwarder",
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(body)
	})

	// Start MQTT connection (non-blocking)
	go initMQTT()

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
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	if sess.Get("authenticated") != true {
		return c.Redirect("/login")
	}

	return c.Next()
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
				fileInfos[name] = FileInfo{Name: name, URL: "/files/" + name, UploadTime: info.ModTime()}
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
						Min:          oldInfo.Min,
						Max:          oldInfo.Max,
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
				if s, ex := m["state"]; ex {
					info.Status = fmt.Sprintf("%v", s)
				} else {
					info.Status = fmt.Sprintf("%v", tmp)
				}
				if mod, ex := m["model"]; ex {
					info.Model = fmt.Sprintf("%v", mod)
				}
				if v, ex := m["ver"]; ex {
					info.Version = fmt.Sprintf("%v", v)
				}
				if ip, ex := m["ip"]; ex {
					info.IP = fmt.Sprintf("%v", ip)
				}
				if p, ex := m["prefix"]; ex {
					info.Prefix = fmt.Sprintf("%v", p)
				}
				// Handle nested "conf" object if present
				if conf, ex := m["conf"]; ex {
					if cm, ok := conf.(map[string]interface{}); ok {
						if v, ok := cm["ck"]; ok {
							info.Ck = fmt.Sprintf("%v", v)
						}
						if v, ok := cm["area"]; ok {
							info.Area = fmt.Sprintf("%v", v)
						}
						if v, ok := cm["no"]; ok {
							info.No = fmt.Sprintf("%v", v)
						}
						if v, ok := cm["min"]; ok {
							if f, ok := v.(float64); ok {
								info.Min = f
							}
						}
						if v, ok := cm["max"]; ok {
							if f, ok := v.(float64); ok {
								info.Max = f
							}
						}
						// Multi-thresholds
						for i := 0; i < 5; i++ {
							minKey := fmt.Sprintf("min%d", i)
							maxKey := fmt.Sprintf("max%d", i)
							if v, ok := cm[minKey]; ok {
								if f, ok := v.(float64); ok {
									switch i {
									case 0:
										info.Min0 = f
									case 1:
										info.Min1 = f
									case 2:
										info.Min2 = f
									case 3:
										info.Min3 = f
									case 4:
										info.Min4 = f
									}
								}
							}
							if v, ok := cm[maxKey]; ok {
								if f, ok := v.(float64); ok {
									switch i {
									case 0:
										info.Max0 = f
									case 1:
										info.Max1 = f
									case 2:
										info.Max2 = f
									case 3:
										info.Max3 = f
									case 4:
										info.Max4 = f
									}
								}
							}
						}
						if v, ok := cm["prefix"]; ok {
							info.Prefix = fmt.Sprintf("%v", v)
						}
						if v, ok := cm["interval"]; ok {
							if f, ok := v.(float64); ok {
								info.Interval = uint64(f)
							}
						}
						// Proximity NC/NO
						for i := 0; i < 3; i++ {
							key := fmt.Sprintf("prox_nc%d", i)
							if v, ok := cm[key]; ok {
								if f, ok := v.(float64); ok {
									switch i {
									case 0:
										info.ProxNc0 = int(f)
									case 1:
										info.ProxNc1 = int(f)
									case 2:
										info.ProxNc2 = int(f)
									}
								}
							}
							// Troli specific fields
							if v, ok := cm["app_mode"]; ok {
								info.AppMode = fmt.Sprintf("%v", v)
							}
							if v, ok := cm["trans"]; ok {
								info.Trans = fmt.Sprintf("%v", v)
							}
							if v, ok := cm["pass_code"]; ok {
								info.PassCode = fmt.Sprintf("%v", v)
							}
						}
					}
				} else {
					info.Status = fmt.Sprintf("%v", tmp)
				}
			} else {
				info.Status = string(raw)
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
			if v, ok := m["ram"]; ok {
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
			if v, ok := m["sd_ok"]; ok {
				if b, ok := v.(bool); ok {
					info.SD_OK = &b
				}
			}
			if v, ok := m["model"]; ok {
				info.Model = fmt.Sprintf("%v", v)
			}
			if v, ok := m["prefix"]; ok {
				info.Prefix = fmt.Sprintf("%v", v)
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
