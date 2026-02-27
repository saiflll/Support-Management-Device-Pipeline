package forwarder

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"IoTT/internal/database"
	"IoTT/internal/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"
)

// --- Configuration ---
const (
	aggregationInterval = 8 * time.Minute
	maxBufferSize       = 50 * 1024 // 50 KB
)

// ForwarderStatus menampung data untuk ditampilkan di dashboard.
type ForwarderStatus struct {
	Topic              string            `json:"topic"`
	ReceivedDataBuffer []models.AreaData `json:"ReceivedDataBuffer"`
	Pipelines          []PipelineStatus  `json:"pipelines"`
}

type Pipeline struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	SourceTopic     string `json:"source_topic"`
	BrokerURL       string `json:"broker_url"`
	DestTopic       string `json:"dest_topic"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	IntervalMinutes int    `json:"interval_minutes"`
	IsActive        bool   `json:"is_active"`
}

type PipelineStatus struct {
	ID                int       `json:"id"`
	SourceTopic       string    `json:"source_topic"`
	LastForwardTime   time.Time `json:"last_forward_time"`
	NextForwardTime   time.Time `json:"next_forward_time"`
	BufferSize        int       `json:"buffer_size"`
	BufferItemCount   int       `json:"buffer_item_count"`
	LastForwardStatus string    `json:"last_forward_status"`
	LastForwardError  string    `json:"last_forward_error"`
	BrokerURL         string    `json:"broker_url"`
	DestTopic         string    `json:"dest_topic"`
}

type PipelineInstance struct {
	Config   Pipeline
	Client   mqtt.Client
	Buffer   []models.AreaData
	Mutex    sync.Mutex
	Ticker   *time.Ticker
	StopChan chan struct{}
	Status   PipelineStatus
}

var (
	// pipelines runtime
	pipelineInstances = make(map[int]*PipelineInstance)
	pipelinesMutex    = &sync.Mutex{}

	status      = ForwarderStatus{}
	statusMutex = &sync.Mutex{}

	// Session store for authentication
	store            *session.Store
	telegramBotToken string
	telegramChatID   string
)

// Start initializes the forwarder component.
func Start() {
	// Initialize session store for authentication
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

	// Load Telegram config from environment
	telegramBotToken = os.Getenv("TELE_BOT_ALRT")
	telegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	if telegramBotToken == "" || telegramChatID == "" {
		log.Println("⚠️ Warning: TELE_BOT_ALRT or TELEGRAM_CHAT_ID not set. Login feature may not work.")
	}

	// Start the pipeline manager
	go pipelineManager()

	log.Println("✅ Forwarder worker started with Dynamic Pipeline Management.")
}

func pipelineManager() {
	for {
		syncPipelines()
		time.Sleep(30 * time.Second)
	}
}

func syncPipelines() {
	importDB := database.DB
	if importDB == nil {
		return
	}

	rows, err := importDB.Query("SELECT pipeline_id, name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active FROM pipelines")
	if err != nil {
		log.Printf("Error querying pipelines: %v", err)
		return
	}
	defer rows.Close()

	foundIDs := make(map[int]bool)
	for rows.Next() {
		var p Pipeline
		var name, username, password sql.NullString
		if err := rows.Scan(&p.ID, &name, &p.SourceTopic, &p.BrokerURL, &p.DestTopic, &username, &password, &p.IntervalMinutes, &p.IsActive); err != nil {
			log.Printf("Error scanning pipeline rows: %v", err)
			continue
		}
		if name.Valid {
			p.Name = name.String
		}
		if username.Valid {
			p.Username = username.String
		}
		if password.Valid {
			p.Password = password.String
		}
		foundIDs[p.ID] = true

		if p.IsActive {
			pipelinesMutex.Lock()
			instance, exists := pipelineInstances[p.ID]
			if !exists {
				log.Printf("🚀 Starting new pipeline: %s -> %s", p.SourceTopic, p.DestTopic)
				instance = startPipeline(p)
				pipelineInstances[p.ID] = instance
			} else if instance.Config != p {
				log.Printf("🔄 Restarting pipeline due to config change: %d", p.ID)
				instance.Stop()
				instance = startPipeline(p)
				pipelineInstances[p.ID] = instance
			}
			pipelinesMutex.Unlock()
		} else {
			pipelinesMutex.Lock()
			if instance, exists := pipelineInstances[p.ID]; exists {
				log.Printf("🛑 Stopping inactive pipeline: %d", p.ID)
				instance.Stop()
				delete(pipelineInstances, p.ID)
			}
			pipelinesMutex.Unlock()
		}
	}

	// Stop pipelines not in DB anymore
	pipelinesMutex.Lock()
	for id, instance := range pipelineInstances {
		if !foundIDs[id] {
			log.Printf("🗑️ Stopping deleted pipeline: %d", id)
			instance.Stop()
			delete(pipelineInstances, id)
		}
	}
	pipelinesMutex.Unlock()
}

func startPipeline(p Pipeline) *PipelineInstance {
	interval := time.Duration(p.IntervalMinutes) * time.Minute
	if interval < 1*time.Minute {
		interval = 1 * time.Minute
	}

	inst := &PipelineInstance{
		Config:   p,
		StopChan: make(chan struct{}),
		Buffer:   []models.AreaData{},
		Status: PipelineStatus{
			ID:                p.ID,
			SourceTopic:       p.SourceTopic,
			BrokerURL:         p.BrokerURL,
			DestTopic:         p.DestTopic,
			LastForwardStatus: "Initializing",
			NextForwardTime:   time.Now().Add(interval),
		},
	}

	// Setup MQTT Client for this pipeline
	opts := mqtt.NewClientOptions()
	opts.AddBroker(p.BrokerURL)
	opts.SetClientID(fmt.Sprintf("forwarder-%d-%d", p.ID, time.Now().UnixNano()))
	if p.Username != "" {
		opts.SetUsername(p.Username)
		opts.SetPassword(p.Password)
	}

	inst.Client = mqtt.NewClient(opts)
	if token := inst.Client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("❌ Failed to connect pipeline %d: %v", p.ID, token.Error())
		inst.Status.LastForwardError = token.Error().Error()
		inst.Status.LastForwardStatus = "Connection Failed"
	}

	inst.Ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-inst.Ticker.C:
				inst.Flush(true)
			case <-inst.StopChan:
				if inst.Client != nil && inst.Client.IsConnected() {
					inst.Client.Disconnect(250)
				}
				inst.Ticker.Stop()
				return
			}
		}
	}()

	return inst
}

func (inst *PipelineInstance) Stop() {
	close(inst.StopChan)
}

func (inst *PipelineInstance) Flush(force bool) {
	inst.Mutex.Lock()
	defer inst.Mutex.Unlock()

	if len(inst.Buffer) == 0 {
		inst.Status.NextForwardTime = time.Now().Add(time.Duration(inst.Config.IntervalMinutes) * time.Minute)
		return
	}

	currentSize := 0
	jsonData, _ := json.Marshal(inst.Buffer)
	currentSize = len(jsonData)

	// Flush if forced or size reached
	if force || currentSize >= maxBufferSize {
		if inst.Client == nil || !inst.Client.IsConnected() {
			log.Printf("Error: MQTT client for pipeline %d not connected. Retrying...", inst.Config.ID)
			inst.Client.Connect() // Try reconnect
			if !inst.Client.IsConnected() {
				inst.Status.LastForwardStatus = "Gagal"
				inst.Status.LastForwardError = "Client disconnected"
				return
			}
		}

		token := inst.Client.Publish(inst.Config.DestTopic, 1, false, jsonData)
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			inst.Status.LastForwardStatus = "Gagal"
			inst.Status.LastForwardError = token.Error().Error()
		} else {
			inst.Status.LastForwardStatus = "Sukses"
			inst.Status.LastForwardError = ""
			inst.Status.LastForwardTime = time.Now()
			inst.Buffer = []models.AreaData{}
		}
		inst.Status.NextForwardTime = time.Now().Add(time.Duration(inst.Config.IntervalMinutes) * time.Minute)
		if inst.Ticker != nil {
			inst.Ticker.Reset(time.Duration(inst.Config.IntervalMinutes) * time.Minute)
		}
	}

	inst.Status.BufferSize = currentSize
	inst.Status.BufferItemCount = len(inst.Buffer)
}

func AddToBufferAndAggregate(data []models.AreaData) {
	// 1. Update global Log Buffer
	statusMutex.Lock()
	status.ReceivedDataBuffer = append(status.ReceivedDataBuffer, data...)
	if len(status.ReceivedDataBuffer) > 100 {
		status.ReceivedDataBuffer = status.ReceivedDataBuffer[len(status.ReceivedDataBuffer)-100:]
	}
	statusMutex.Unlock()

	// 2. Distribute to Pipeline Buffers
	pipelinesMutex.Lock()
	defer pipelinesMutex.Unlock()

	for _, inst := range pipelineInstances {
		inst.Mutex.Lock()
		for _, d := range data {
			// SIMPLE TOPIC MATCHING
			if inst.Config.SourceTopic == "ALL" || d.Topic == inst.Config.SourceTopic {
				if len(inst.Buffer) < 2000 {
					// Create copy and strip topic for clean forwarding (Format 4)
					cleanData := d
					cleanData.Topic = ""
					inst.Buffer = append(inst.Buffer, cleanData)
				}
			}
		}
		inst.Mutex.Unlock()
		inst.Flush(false) // Check if size limit reached
	}
}

// RegisterForwarderHandlers mendaftarkan rute HTTP untuk dashboard.
func RegisterForwarderHandlers(app *fiber.App) {
	// Public routes - Login
	app.Get("/login", handleShowLogin)
	app.Post("/login", handleLogin)
	app.Post("/request-code", handleRequestCode)

	// Protected routes (AUTH DISABLED)
	app.Get("/forwarder", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Forwarder Status",
		})
	})

	// Rute untuk API status (AUTH DISABLED)
	app.Get("/forwarder/status", func(c *fiber.Ctx) error {
		statusMutex.Lock()
		defer statusMutex.Unlock()

		pipelinesMutex.Lock()
		status.Pipelines = []PipelineStatus{}
		for _, inst := range pipelineInstances {
			status.Pipelines = append(status.Pipelines, inst.Status)
		}
		pipelinesMutex.Unlock()

		return c.Status(http.StatusOK).JSON(status)
	})

	// CRUD Pipeline API (AUTH DISABLED for now)
	app.Get("/api/pipelines", func(c *fiber.Ctx) error {
		importDB := database.DB
		rows, _ := importDB.Query("SELECT pipeline_id, name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active FROM pipelines")
		defer rows.Close()
		var res []Pipeline
		for rows.Next() {
			var p Pipeline
			var name, username, password sql.NullString
			rows.Scan(&p.ID, &name, &p.SourceTopic, &p.BrokerURL, &p.DestTopic, &username, &password, &p.IntervalMinutes, &p.IsActive)
			if name.Valid {
				p.Name = name.String
			}
			if username.Valid {
				p.Username = username.String
			}
			if password.Valid {
				p.Password = password.String
			}
			res = append(res, p)
		}
		return c.JSON(res)
	})

	app.Post("/api/pipelines", func(c *fiber.Ctx) error {
		var p Pipeline
		if err := c.BodyParser(&p); err != nil {
			return err
		}
		importDB := database.DB
		err := importDB.QueryRow("INSERT INTO pipelines (name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING pipeline_id",
			p.Name, p.SourceTopic, p.BrokerURL, p.DestTopic, p.Username, p.Password, p.IntervalMinutes, p.IsActive).Scan(&p.ID)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.JSON(p)
	})

	app.Delete("/api/pipelines/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		importDB := database.DB
		importDB.Exec("DELETE FROM pipelines WHERE pipeline_id = $1", id)
		return c.SendStatus(204)
	})

	// Rute untuk redirect ke Database (protected)
	app.Get("/database", requireAuth, func(c *fiber.Ctx) error {
		// Redirect to pgweb on localhost
		return c.Redirect("http://localhost:8080", http.StatusFound)
	})

	// Logout endpoint
	app.Post("/logout", handleLogout)
}

// --- Authentication Handlers ---

func requireAuth(c *fiber.Ctx) error {
	// TEMPORARY: Bypass authentication for debugging/setup
	return c.Next()

	/* -- Original Authentication Logic --
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
	// Bypass forwarder login
	return c.Redirect("/forwarder")
}

func handleLogin(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	submittedCode := c.FormValue("code")
	if submittedCode == "" {
		return c.Render("login", fiber.Map{"error": "Kode tidak boleh kosong."})
	}

	// Get code from session
	authCode := sess.Get("auth_code")
	authExpires := sess.Get("auth_expires")

	if authCode == nil || authExpires == nil {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

	// Convert Unix timestamp back to time.Time
	expiryUnix, ok := authExpires.(int64)
	if !ok {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

	expiryTime := time.Unix(expiryUnix, 0)

	if authCode.(string) != submittedCode || time.Now().After(expiryTime) {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

	// Code is valid, clear from session and set login status
	sess.Delete("auth_code")
	sess.Delete("auth_expires")
	sess.Set("authenticated", true)
	if err := sess.Save(); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Gagal menyimpan sesi")
	}

	return c.Redirect("/forwarder")
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

	// Generate random 6 digit code
	b := make([]byte, 3)
	rand.Read(b)
	code := hex.EncodeToString(b)

	// Save code and expiry as Unix timestamp (int64)
	sess.Set("auth_code", code)
	sess.Set("auth_expires", time.Now().Add(5*time.Minute).Unix())

	if err := sess.Save(); err != nil {
		log.Printf("Error saving session: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": fmt.Sprintf("Gagal menyimpan sesi: %v", err)})
	}

	log.Printf("Code generated and saved: %s", code)

	// Send code to Telegram
	message := fmt.Sprintf("Kode verifikasi Anda untuk Forwarder Dashboard adalah: `%s`\nKode ini berlaku selama 5 menit.", code)
	go sendTelegramMessage(message)

	return c.JSON(fiber.Map{"status": "ok"})
}

func handleLogout(c *fiber.Ctx) error {
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}
	sess.Destroy()
	return c.Redirect("/login")
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
