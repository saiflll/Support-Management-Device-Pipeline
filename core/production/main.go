package main

import (
	"production/modul/mdcw"
	"production/modul/sp"

	"github.com/gofiber/fiber/v2"
)

func main() {
	atrLogger()

	// Google Sheets export has been disabled as requested.

	initDB()
	defer closeDB()

	// Mulai background scraper
	go StartScraper()

	mqttClient := initMQTT()

	mdcw.Init(db, mqttClient)
	mdcw.InitCloudForwarder()

	sp.Init(db, mqttClient)

	ap := fiber.New()
	setupRoutes(ap)

	ftl("%v", ap.Listen(":3000"))
}
