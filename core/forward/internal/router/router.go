package router

import (
	"github.com/gofiber/fiber/v2"

	"IoTT/internal/handler"
)

func SetupInternalRouter(app *fiber.App) {

	dataRoutes := app.Group("/data")
	{

		dataRoutes.Post("", handler.HandleSensorData)
		dataRoutes.Post("/compress", handler.HandleCompress)

	}

	telegramRoutes := app.Group("/telegram")
	{

		telegramRoutes.Post("/webhook", handler.HandleTelegramWebhook)
	}

	app.Get("/monitor/status", handler.MonitorStatusHandler)
}
