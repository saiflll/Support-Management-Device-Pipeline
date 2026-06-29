package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	atrLogger()
	lg("🚀 Starting Produksi/Conveyor Monitoring Service...")

	initDB()
	defer closeDB()

	app := fiber.New(fiber.Config{
		AppName: "Produksi Conveyor Monitor API",
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	// ── Status ────────────────────────────────────────
	app.Get("/api/db-status", handleDBStatus)

	// ── Data Table ────────────────────────────────────
	app.Get("/api/conveyor/latest", handleLatest)
	app.Get("/api/conveyor/range", handleRange)

	// ── Analytics (baru) ──────────────────────────────
	app.Get("/api/analytics/summary-today", handleSummaryToday)
	app.Get("/api/analytics/weekly", handleWeeklyProduction)
	app.Get("/api/analytics/product-dist", handleProductDistribution)
	app.Get("/api/analytics/shift-timeline", handleShiftTimeline)

	// ── Static UI ─────────────────────────────────────
	app.Static("/", "./web")
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})

	port := getEnv("PORT", "3001")
	lg("📡 Running Fiber server on port :%s", port)
	if err := app.Listen(":" + port); err != nil {
		ftl("Failed to start Fiber: %v", err)
	}
}

func handleDBStatus(c *fiber.Ctx) error {
	status := "connected"
	var details string
	if db == nil {
		status = "disconnected"
		details = "Database connection not initialized"
	} else if err := db.Ping(); err != nil {
		status = "error"
		details = err.Error()
	}
	return c.JSON(fiber.Map{"status": status, "details": details})
}

func handleLatest(c *fiber.Ctx) error {
	data, cols, err := getLatestData()
	if err != nil {
		hndlErr("handleLatest", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal mengambil data terbaru", "details": err.Error()})
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"columns": cols, "data": data})
}

func handleRange(c *fiber.Ctx) error {
	start := c.Query("start")
	end := c.Query("end")
	if start == "" || end == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Parameter 'start' dan 'end' wajib diisi (format YYYY-MM-DD)"})
	}
	data, cols, err := getDataByRange(start, end)
	if err != nil {
		hndlErr("handleRange", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal mengambil data range", "details": err.Error()})
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"columns": cols, "data": data})
}

// ── Analytics Handlers ────────────────────────────────────────────

func handleSummaryToday(c *fiber.Ctx) error {
	summary, err := getSummaryToday()
	if err != nil {
		hndlErr("handleSummaryToday", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}

func handleWeeklyProduction(c *fiber.Ctx) error {
	data, err := getWeeklyProduction()
	if err != nil {
		hndlErr("handleWeeklyProduction", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func handleProductDistribution(c *fiber.Ctx) error {
	data, err := getProductDistribution()
	if err != nil {
		hndlErr("handleProductDistribution", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func handleShiftTimeline(c *fiber.Ctx) error {
	date := c.Query("date")
	if date == "" {
		// default: hari ini
		date = timeNow()
	}
	data, err := getShiftTimeline(date)
	if err != nil {
		hndlErr("handleShiftTimeline", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}
