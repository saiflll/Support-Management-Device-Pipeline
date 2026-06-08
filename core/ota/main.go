package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"

	"iot-ota-server/core"
)

func main() {
	core.AtrLogger()

	targetDir := filepath.Join("static", "uploads")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		core.Ftl("failed to create upload directory: %v", err)
	}

	core.LoadInitialFiles(targetDir)

	core.TelegramBotToken = core.GetEnv("TELE_BOT_OTA", "")
	core.TelegramChatID = core.GetEnv("TELEGRAM_CHAT_ID", "")
	core.WebhookToken = core.GetEnv("WEBHOOK_TOKEN", "")
	if core.TelegramBotToken == "" || core.TelegramChatID == "" {
		core.Lg("Peringatan: TELE_BOT_OTA atau TELEGRAM_CHAT_ID tidak diatur. Fitur login tidak akan berfungsi.")
	}

	core.Store = session.New(session.Config{
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

	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
	})

	app.Static("/files", "./static/uploads")

	// public routes
	app.Post("/api/webhook/emqx", core.HandleEMQXWebhook)
	app.Get("/login", core.HandleShowLogin)
	app.Post("/login", core.HandleLogin)
	app.Post("/request-code", core.HandleRequestCode)

	// protected routes
	protected := app.Group("/")
	protected.Use(core.RequireAuth)

	protected.Post("/logout", core.HandleLogout)

	// api routes
	api := protected.Group("/api")
	api.Get("/nodes", core.HandleGetNodes)
	api.Get("/models", core.HandleGetModels)
	api.Get("/models/:name", core.HandleGetModelByName)
	api.Get("/nodes/:id/config", core.HandleGetNodeConfig)
	api.Delete("/nodes/:id", core.HandleDeleteNode)
	api.Get("/files", core.HandleGetFiles)
	api.Delete("/files/*", core.HandleDeleteFile)
	api.Post("/files/:name/rename", core.HandleRenameFile)
	api.Post("/upload", core.HandleUpload)
	api.Post("/config", core.HandlePostConfig)
	api.Post("/ota", core.HandlePostOTA)
	api.Post("/reboot", core.HandlePostReboot)
	api.Get("/logs/:id", core.HandleGetLogs)
	api.Get("/forwarder/status", core.HandleGetForwarderStatus)
	api.Get("/monitor/status", core.HandleGetMonitorStatus)
	api.Get("/pipelines", core.HandleGetPipelines)
	api.Post("/pipelines", core.HandlePostPipelines)
	api.Delete("/pipelines/:id", core.HandleDeletePipeline)

	go core.InitMQTT()

	protected.Static("/", "./web")
	protected.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})

	// offline nodes cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			core.NodeMutex.Lock()
			now := time.Now()
			for id, info := range core.NodeStatus {
				if info.Updated != "" {
					updatedTime, err := time.Parse("2006-01-02 15:04:05", info.Updated)
					if err == nil && now.Sub(updatedTime) > 7*24*time.Hour {
						core.Lg("Cleaning up old node: %s", id)
						delete(core.NodeStatus, id)
					}
				}
			}
			core.NodeMutex.Unlock()
		}
	}()

	core.Ftl("%v", app.Listen("0.0.0.0:9999"))
}
