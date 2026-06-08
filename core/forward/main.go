package main

import (
	"IoTT/internal/archiver"
	"IoTT/internal/config"
	"IoTT/internal/database"
	"IoTT/internal/forwarder"
	"IoTT/internal/logger"
	"IoTT/internal/models"
	"IoTT/internal/mqtt"
	internalrouter "IoTT/internal/router"
	"IoTT/internal/telegram"
	"os"
	_ "time/tzdata" // Import untuk menyematkan database zona waktu

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
)

// === ENTRYPOINT UTAMA ===

func main() {
	// inisialisasi logger untuk mode debug
	logger.AtrLogger()

	// inisialisasi zona waktu aplikasi ke Asia/Jakarta (UTC+7)
	config.InitTimezone()

	database.InitDB()
	if database.DB != nil {
		defer database.CloseDB()
	}

	telegram.LoadConfig()
	if err := telegram.InitBot(); err != nil {
		logger.HndlErr("Gagal menginisialisasi bot Telegram. Notifikasi mungkin tidak berfungsi.", err)
	}

	// jalankan MQTT client di goroutine agar tidak memblokir server HTTP
	go mqtt.StartClient()

	// memulai worker yang menjalankan pengecekan periodik (sensor offline, pintu terbuka, dll.)
	models.StartPeriodicCheckWorker()

	// memulai worker untuk arsip data lama
	go archiver.Start()

	// memulai worker untuk forwarder ke EMQX Publik
	go forwarder.Start()

	// memuat ulang data lookup untuk memastikan semua data hasil seeding tersedia.
	logger.Lg("🔄 Memuat ulang data lookup (Area & Pintu)...")
	database.LoadLookupData()

	engine := html.New("./internal/forwarder", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// middleware
	app.Use(cors.New()) // Tambahkan CORS untuk pengembangan

	// daftarkan handler untuk dashboard forwarder
	forwarder.RegisterForwarderHandlers(app)

	internalrouter.SetupInternalRouter(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}
	logger.Lg("Start server Fiber: %s", port)

	if err := app.Listen(":" + port); err != nil {
		logger.Ftl("Gagal menjalankan server Fiber: %v", err)
	}
}
