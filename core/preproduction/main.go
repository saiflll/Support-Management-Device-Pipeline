package main

import (
	"preproduction/lib"
	"preproduction/modul/mdcw"
	"preproduction/modul/sp"

	"github.com/gofiber/fiber/v2"
)

func main() {
	atrLogger()

	if err := lib.InitGoogleSheets(); err != nil {
		hndlErr("Google Sheets initialization failed", err)
	} else {
		lib.CreateSheetIfNotExists()
	}

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
