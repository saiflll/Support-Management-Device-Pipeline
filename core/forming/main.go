package main

import (
	"forming/lib"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	if err := lib.InitGoogleSheets(); err != nil {
		log.Printf("Warning: Google Sheets initialization failed: %v", err)
	} else {
		lib.CreateSheetIfNotExists()
	}

	initDB()
	defer closeDB()

	initMQTT()

	app := fiber.New()
	setupRoutes(app)
	
	log.Fatal(app.Listen(":3000"))
}
