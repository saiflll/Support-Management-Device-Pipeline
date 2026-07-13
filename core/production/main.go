package main

import (
	"production/modul/mdcw"
	"production/modul/sp"
	"production/redis"

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

	// Initialize Redis (optional — graceful if REDIS_URL not set)
	redisClient := redis.InitRedis()
	if redisClient != nil {
		// Subscribe to sensor status changes for health monitoring
		redis.SubscribeSensorStatus(func(msg map[string]interface{}) {
			if sk, ok := msg["sensor_key"]; ok {
				if st, ok := msg["status"]; ok {
					lg("[Redis] Sensor status: %s → %s", sk, st)
				}
			}
		})
		lg("✅ Redis integration initialized")
	}

	mdcw.Init(db, mqttClient)
	mdcw.InitCloudForwarder()

	sp.Init(db, mqttClient)

	ap := fiber.New()
	setupRoutes(ap)

	ftl("%v", ap.Listen(":3000"))
}
