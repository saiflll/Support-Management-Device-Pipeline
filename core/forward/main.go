package main

import (
	"IoTT/internal/archiver"
	"IoTT/internal/config"
	"IoTT/internal/database"
	"IoTT/internal/logger"
	"IoTT/internal/models"
	"IoTT/internal/mqtt"
	"IoTT/internal/redis"
	internalrouter "IoTT/internal/router"
	"IoTT/internal/telegram"
	"os"
	_ "time/tzdata" // Import untuk menyematkan database zona waktu

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Inisialisasi Redis (opsional — jika REDIS_URL tidak diset, Redis tidak dipakai)
	redis.InitRedis()

	telegram.LoadConfig()
	if err := telegram.InitBot(); err != nil {
		logger.HndlErr("Gagal menginisialisasi bot Telegram. Notifikasi mungkin tidak berfungsi.", err)
	}

	// jalankan MQTT client
	go mqtt.StartClient()

	// memulai worker yang menjalankan pengecekan periodik (sensor offline, pintu terbuka, dll.)
	models.StartPeriodicCheckWorker()

	// memulai worker untuk arsip data lama
	go archiver.Start()

	// memuat ulang data lookup untuk memastikan semua data hasil seeding tersedia.
	logger.Lg("🔄 Memuat ulang data lookup (Area & Pintu)...")
	database.LoadLookupData()

	app := fiber.New(fiber.Config{})

	// middleware
	app.Use(cors.New())

	// health check (untuk Docker HEALTHCHECK)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "UP"})
	})

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
