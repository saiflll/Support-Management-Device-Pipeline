package main

import (
	"forming/lib"

	"github.com/gofiber/fiber/v2"
)

// === MAIN ENTRYPOINT ===

func main() {
	// inisialisasi logger untuk mode debug
	atrLogger()

	if err := lib.InitGoogleSheets(); err != nil {
		hndlErr("Google Sheets initialization failed", err)
	} else {
		lib.CreateSheetIfNotExists()
	}

	initDB()
	defer closeDB()

	// inisialisasi MQTT lokal (subscribe dari perangkat IoT)
	initMQTT()

	// inisialisasi Cloud MQTT Forwarder (publish ke backend cloud)
	initCloudForwarder()

	ap := fiber.New()
	setupRoutes(ap)

	ftl("%v", ap.Listen(":3000"))
}
