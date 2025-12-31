package forwarder

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
	"sync"
	"time"

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
	LastForwardTime    time.Time
	NextForwardTime    time.Time
	BufferSize         int
	BufferItemCount    int
	LastForwardStatus  string
	LastForwardError   string
	ReceivedDataBuffer []models.AreaData
}

var (
	buffer           []models.AreaData
	bufferMutex      = &sync.Mutex{}
	publicMqttClient mqtt.Client
	status           = ForwarderStatus{
		LastForwardStatus: "Belum ada",
	}
	statusMutex = &sync.Mutex{}
	ticker      *time.Ticker

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

	setupPublicMQTT()
	ticker = time.NewTicker(aggregationInterval)

	statusMutex.Lock()
	status.NextForwardTime = time.Now().Add(aggregationInterval)
	statusMutex.Unlock()

	go func() {
		for {
			<-ticker.C
			log.Println("⏰ (Forwarder) Timer 8 menit tercapai, memicu proses rekap data.")
			flushBufferIfNecessary(true) // Force flush on timer
		}
	}()

	log.Println("✅ Forwarder worker started. Will aggregate data and forward to public EMQX.")
}

func AddToBufferAndAggregate(data []models.AreaData) {
	bufferMutex.Lock()
	// Limit buffer to 2000 items (approx 2 MB) to prevent OOM
	if len(buffer) < 2000 {
		buffer = append(buffer, data...)
	} else {
		log.Println("⚠️ Warning: Forwarder local buffer full (2000 items). Dropping new data to prevent OOM.")
	}
	bufferMutex.Unlock()

	// Update status for dashboard display
	statusMutex.Lock()
	status.ReceivedDataBuffer = append(status.ReceivedDataBuffer, data...)
	if len(status.ReceivedDataBuffer) > 100 { // Keep only last 100 items for display
		status.ReceivedDataBuffer = status.ReceivedDataBuffer[len(status.ReceivedDataBuffer)-100:]
	}
	statusMutex.Unlock()

	flushBufferIfNecessary(false) // Check if flush is needed due to size
}

func flushBufferIfNecessary(force bool) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()

	currentSize := 0
	if len(buffer) > 0 {
		jsonData, _ := json.Marshal(buffer)
		currentSize = len(jsonData)
	}

	// Flush if forced, time is up, or size limit is reached
	if len(buffer) > 0 && (force || time.Now().After(status.NextForwardTime) || currentSize >= maxBufferSize) {
		log.Printf("Flushing buffer. Items: %d, Size: %d, Forced: %v", len(buffer), currentSize, force)
		forwardData(buffer)
		buffer = []models.AreaData{} // Clear buffer

		statusMutex.Lock()
		status.NextForwardTime = time.Now().Add(aggregationInterval)
		// Don't clear status.ReceivedDataBuffer here if we want to see it on the web
		// status.ReceivedDataBuffer = []models.AreaData{}
		ticker.Reset(aggregationInterval) // Reset timer
		statusMutex.Unlock()
	}

	// Always update buffer size and count for the dashboard
	statusMutex.Lock()
	status.BufferSize = currentSize
	status.BufferItemCount = len(buffer)
	statusMutex.Unlock()
}

func forwardData(dataToForward []models.AreaData) {
	statusMutex.Lock()
	status.LastForwardTime = time.Now()
	statusMutex.Unlock()

	if publicMqttClient == nil || !publicMqttClient.IsConnected() {
		log.Println("Error: Public MQTT client not connected. Skipping forward.")
		statusMutex.Lock()
		status.LastForwardStatus = "Gagal"
		status.LastForwardError = "Client tidak terhubung"
		statusMutex.Unlock()
		return
	}

	topic := os.Getenv("MQTT_TOPIC_PUB")
	if topic == "" {
		topic = "sensor/data/ingest" // Default topic
	}

	payload, err := json.Marshal(dataToForward)
	if err != nil {
		log.Printf("Error marshalling data for forwarding: %v", err)
		statusMutex.Lock()
		status.LastForwardStatus = "Gagal"
		status.LastForwardError = fmt.Sprintf("JSON Marshal Error: %v", err)
		statusMutex.Unlock()
		return
	}

	token := publicMqttClient.Publish(topic, 1, false, payload)
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		log.Printf("Error forwarding data to public MQTT: %v", token.Error())
		statusMutex.Lock()
		status.LastForwardStatus = "Gagal"
		status.LastForwardError = fmt.Sprintf("Publish Error: %v", token.Error())
		statusMutex.Unlock()
	} else {
		log.Printf("Successfully forwarded %d data points to topic %s", len(dataToForward), topic)
		statusMutex.Lock()
		status.LastForwardStatus = "Sukses"
		status.LastForwardError = ""
		statusMutex.Unlock()
	}
}

func setupPublicMQTT() {
	broker := os.Getenv("MQTT_BROKER_PUB")
	if broker == "" {
		log.Println("Warning: MQTT_BROKER_PUB not set. Forwarder will not work.")
		return
	}
	clientID := fmt.Sprintf("servfi-forwarder-%d", time.Now().UnixNano())

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.OnConnect = func(c mqtt.Client) { log.Println("✅ Forwarder connected to Public MQTT Broker.") }
	opts.OnConnectionLost = func(c mqtt.Client, err error) { log.Printf("⚠️ Forwarder connection to Public MQTT lost: %v", err) }

	// Tambahkan kredensial jika tersedia di environment
	username := os.Getenv("MQTT_USERNAME_FOR")
	password := os.Getenv("MQTT_PASSWORD_FOR")
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
		log.Println("(Forwarder) Menggunakan kredensial MQTT.")
	}

	publicMqttClient = mqtt.NewClient(opts)
	if token := publicMqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("❌ Failed to connect forwarder to public MQTT broker: %v", token.Error())
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
		return c.Status(http.StatusOK).JSON(status)
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
