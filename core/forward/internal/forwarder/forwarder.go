package forwarder

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"IoTT/internal/auth"
	"IoTT/internal/database"
	"IoTT/internal/models"
	"IoTT/internal/worker"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
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

	// Telegram config untuk notifikasi
	telegramBotToken string
	telegramChatID   string

	// [FIX] Deadband State Tracking
	// Key format: "ck-area-sensorNo"
	lastSensorValues = make(map[string]float64)
	lastDoorStatus   = make(map[string]int)
	lastSentTime     = make(map[string]time.Time)
	deltaMutex       = &sync.Mutex{}
)

// Start menginisialisasi forwarder component.
func Start() {
	// Inisialisasi session store untuk autentikasi
	auth.InitAuth()

	// Load Telegram config dari environment
	telegramBotToken = os.Getenv("TELE_BOT_ALRT")
	telegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	if telegramBotToken == "" || telegramChatID == "" {
		log.Println("⚠️ Warning: TELE_BOT_ALRT atau TELEGRAM_CHAT_ID tidak diset. Fitur login via Telegram tidak akan berfungsi.")
	}

	// Mulai pipeline manager
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

	// Stop pipelines yang sudah tidak ada di DB
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

	// Setup MQTT Client untuk pipeline ini
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
	} else {
		inst.Status.LastForwardStatus = "Sukses"
	}

	// Daftarkan client ke RetryWorker agar bisa dipakai untuk re-publish
	worker.RegisterRetryClient(p.ID, inst.Client)

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

	// Flush jika dipaksa atau ukuran buffer sudah penuh
	if force || currentSize >= maxBufferSize {
		if inst.Client == nil || !inst.Client.IsConnected() {
			log.Printf("Error: MQTT client untuk pipeline %d tidak terhubung. Menyimpan ke DB sebagai fallback...", inst.Config.ID)
			worker.SaveFailedBatch(inst.Config.ID, inst.Config.DestTopic, inst.Config.BrokerURL, jsonData)
			inst.Buffer = []models.AreaData{}
			inst.Status.LastForwardStatus = "Gagal (disimpan ke DB)"
			inst.Status.LastForwardError = "Client disconnected"
			inst.Status.NextForwardTime = time.Now().Add(time.Duration(inst.Config.IntervalMinutes) * time.Minute)
			return
		}

		token := inst.Client.Publish(inst.Config.DestTopic, 1, false, jsonData)
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			// Publish gagal → simpan ke DB untuk di-retry nanti
			log.Printf("❌ Pipeline %d publish gagal: %v. Menyimpan ke failed_batch...", inst.Config.ID, token.Error())
			worker.SaveFailedBatch(inst.Config.ID, inst.Config.DestTopic, inst.Config.BrokerURL, jsonData)
			inst.Status.LastForwardStatus = "Gagal (disimpan ke DB)"
			inst.Status.LastForwardError = token.Error().Error()
		} else {
			// Publish berhasil → bersihkan buffer
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
	// 1. Update global Log Buffer untuk dashboard UI
	statusMutex.Lock()
	status.ReceivedDataBuffer = append(status.ReceivedDataBuffer, data...)
	if len(status.ReceivedDataBuffer) > 100 {
		status.ReceivedDataBuffer = status.ReceivedDataBuffer[len(status.ReceivedDataBuffer)-100:]
	}
	statusMutex.Unlock()

	// 2. Filter data untuk Pipeline (Deadband Logic)
	deltaMutex.Lock()
	defer deltaMutex.Unlock()

	for _, areaData := range data {
		filteredData := models.AreaData{
			CK:    areaData.CK,
			Area:  areaData.Area,
			Topic: "",
			Temp:  []models.TempData{},
			Door:  []models.DoorData{},
		}

		// --- Filter Temp ---
		for _, t := range areaData.Temp {
			key := fmt.Sprintf("%d-%d-%d", areaData.CK, areaData.Area, t.No)
			lastVal, exists := lastSensorValues[key]
			lastTs := lastSentTime[key]
			delta := 0.0
			if t.Temp > lastVal {
				delta = t.Temp - lastVal
			} else {
				delta = lastVal - t.Temp
			}

			// Kirim jika: baru pertama, delta > 2.0, atau > 10 menit (heartbeat)
			if !exists || delta >= 2.0 || time.Since(lastTs) > 10*time.Minute {
				filteredData.Temp = append(filteredData.Temp, t)
				lastSensorValues[key] = t.Temp
				lastSentTime[key] = time.Now()
			}
		}

		// --- Filter Door ---
		for _, d := range areaData.Door {
			key := fmt.Sprintf("door-%d-%d-%d", areaData.CK, areaData.Area, d.DoorID)
			lastVal, exists := lastDoorStatus[key]
			if !exists || d.Value != lastVal {
				filteredData.Door = append(filteredData.Door, d)
				lastDoorStatus[key] = d.Value
			}
		}

		// Jika ada data yang lolos filter, masukkan ke pipeline
		if len(filteredData.Temp) > 0 || len(filteredData.Door) > 0 {
			pipelinesMutex.Lock()
			for _, inst := range pipelineInstances {
				if inst.Config.IsActive && (inst.Config.SourceTopic == "ALL" || areaData.Topic == inst.Config.SourceTopic) {
					inst.Mutex.Lock()
					if len(inst.Buffer) < 2000 {
						inst.Buffer = append(inst.Buffer, filteredData)
					}
					inst.Mutex.Unlock()
				}
			}
			pipelinesMutex.Unlock()
		}
	}
}

// RegisterForwarderHandlers mendaftarkan rute HTTP untuk dashboard.
func RegisterForwarderHandlers(app *fiber.App) {
	// Public routes
	app.Get("/login", handleShowLogin)
	app.Post("/login", handleLogin)
	app.Post("/request-code", handleRequestCode)
	app.Post("/logout", handleLogout)

	// Dashboard (auth toggle lewat DEV_MODE env)
	app.Get("/forwarder", auth.RequireAuth, func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Forwarder Status",
		})
	})

	// API status
	app.Get("/forwarder/status", auth.RequireAuth, func(c *fiber.Ctx) error {
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

	// CRUD Pipeline API
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
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Pipeline deleted successfully",
		})
	})

	// Redirect ke pgweb database viewer (protected)
	app.Get("/database", auth.RequireAuth, func(c *fiber.Ctx) error {
		return c.Redirect("http://localhost:8080", http.StatusFound)
	})
}

// --- Authentication Handlers ---

func handleShowLogin(c *fiber.Ctx) error {
	if auth.IsDevMode() {
		return c.Redirect("/forwarder")
	}
	return c.Render("login", fiber.Map{})
}

func handleLogin(c *fiber.Ctx) error {
	if auth.Store == nil {
		auth.InitAuth()
	}
	sess, err := auth.Store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	submittedCode := c.FormValue("code")
	if submittedCode == "" {
		return c.Render("login", fiber.Map{"error": "Kode tidak boleh kosong."})
	}

	authCode := sess.Get("auth_code")
	authExpires := sess.Get("auth_expires")
	if authCode == nil || authExpires == nil {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

	expiryUnix, ok := authExpires.(int64)
	if !ok {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

	if authCode.(string) != submittedCode || time.Now().After(time.Unix(expiryUnix, 0)) {
		return c.Render("login", fiber.Map{"error": "Kode verifikasi salah atau sudah kedaluwarsa."})
	}

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

	if auth.Store == nil {
		auth.InitAuth()
	}
	sess, err := auth.Store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Gagal membuat sesi."})
	}

	// Generate random 6-digit hex code
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Gagal generate kode."})
	}
	code := fmt.Sprintf("%x", b)

	sess.Set("auth_code", code)
	sess.Set("auth_expires", time.Now().Add(5*time.Minute).Unix())
	if err := sess.Save(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": fmt.Sprintf("Gagal menyimpan sesi: %v", err)})
	}

	message := fmt.Sprintf("Kode verifikasi Forwarder Dashboard: `%s`\nBerlaku 5 menit.", code)
	go sendTelegramMessage(message)

	return c.JSON(fiber.Map{"status": "ok"})
}

func handleLogout(c *fiber.Ctx) error {
	if auth.Store == nil {
		return c.Redirect("/login")
	}
	sess, err := auth.Store.Get(c)
	if err != nil {
		return c.Redirect("/login")
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
