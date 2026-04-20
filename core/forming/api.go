package main

import (
	"encoding/csv"
	"fmt"
	"forming/lib"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	jwtSecret  = "ppa3-secret-jwt-2025"
	attendUser = "ppa3"
	attendPass = "plan3ppa"
)

func requireJWT(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if len(token) > 7 && strings.HasPrefix(token, "Bearer ") { token = token[7:] }
	if token == "" { token = c.Cookies("forming_token") }
	if token == "" { return c.Status(401).JSON(fiber.Map{"error": "unauthorized"}) }
	
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !parsed.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}
	return c.Next()
}

// setupRoutes binds endpoints to fiber
func setupRoutes(app *fiber.App) {
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Basic Handlers
	app.Post("/api/login", handleLogin)
	app.Post("/api/logout", handleLogout)

	api := app.Group("/api", requireJWT)
	api.Get("/data", handleGetData)
	api.Get("/summary", handleGetSummary)
	api.Get("/prefixes", handleGetPrefixes)
	api.Get("/skip-log", handleGetSkipLogs)
	api.Get("/export-csv", handleExportCsv)

	app.Static("/", "./web")
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})
}

func handleLogin(c *fiber.Ctx) error {
	type Req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Username != attendUser || req.Password != attendPass {
		return c.Status(401).JSON(fiber.Map{"error": "username atau password salah"})
	}
	
	claims := jwtClaims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}
	
	c.Cookie(&fiber.Cookie{
		Name: "forming_token", Value: token, HTTPOnly: true, SameSite: "Lax", MaxAge: 43200,
	})
	return c.JSON(fiber.Map{"token": token, "username": req.Username})
}

func handleLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{Name: "forming_token", Value: "", MaxAge: -1})
	return c.JSON(fiber.Map{"status": "ok"})
}

func handleGetData(c *fiber.Ctx) error {
	records, err := getRecords(
		c.Query("prefix"), c.Query("status"), c.Query("sort", "newest"), 
		c.Query("start_date"), c.Query("end_date"),
	)
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(records)
}

func handleGetSummary(c *fiber.Ctx) error {
	summaries, err := getSummary()
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(summaries)
}

func handleGetPrefixes(c *fiber.Ctx) error {
	prefixes, err := lib.GetPrefixes(db)
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(prefixes)
}

func handleGetSkipLogs(c *fiber.Ctx) error {
	skipLogs, err := lib.GetSkipLogs(db)
	if err != nil { return c.JSON([]map[string]interface{}{}) }
	return c.JSON(skipLogs)
}

func handleExportCsv(c *fiber.Ctx) error {
	start, end := c.Query("start_date"), c.Query("end_date")
	status, pfx := c.Query("status"), c.Query("prefix")

	pfxName := "ALL"
	if pfx != "" && pfx != "all" { pfxName = strings.ReplaceAll(pfx, " ", "_") }

	filename := fmt.Sprintf("mdcw_export_%s_%s.csv", pfxName, time.Now().Format("20060102"))
	if start != "" && end != "" { filename = fmt.Sprintf("mdcw_%s_%s_to_%s.csv", pfxName, start, end) }

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set("Content-Type", "text/csv")

	records, err := lib.GetRecordsByDateRange(db, start, end, pfx, status, "newest")
	if err != nil { return c.Status(500).SendString(err.Error()) }

	w := csv.NewWriter(c.Response().BodyWriter())
	w.Write([]string{"ID", "Timestamp", "Prefix", "Berat (g)", "Pack Count", "Status", "DataType", "Confidence"})
	
	for _, r := range records {
		st := ""
		switch r.Reg5 {
		case 41, 521, 553: st = "OK"
		case 8: st = "MATI"
		case 9, 90: st = "IDLE"
		case 8201: st = "METAL"
		case 25: st = "UNDER"
		case 73: st = "OVER"
		default: st = fmt.Sprintf("UNKNOWN (%d)", r.Reg5)
		}
		w.Write([]string{
			fmt.Sprintf("%d", r.ID), r.Ts, r.Prefix, r.WeightFormatted, fmt.Sprintf("%d", r.Reg2), st,
			r.DataType, fmt.Sprintf("%.2f", r.Confidence),
		})
	}
	w.Flush()
	return nil
}
