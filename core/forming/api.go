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

// === KONSTANTA & KONFIGURASI ===

const (
	jwtSecret  = "ppa3-secret-jwt-2025"
	attendUser = "ppa3"
	attendPass = "plan3ppa"
)

// === JWT MIDDLEWARE ===

func requireJWT(c *fiber.Ctx) error {
	tkn := c.Get("Authorization")
	if len(tkn) > 7 && strings.HasPrefix(tkn, "Bearer ") { tkn = tkn[7:] }
	if tkn == "" { tkn = c.Cookies("forming_token") }
	if tkn == "" { return c.Status(401).JSON(fiber.Map{"error": "unauthorized"}) }
	
	prs, err := jwt.ParseWithClaims(tkn, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !prs.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}
	return c.Next()
}

// === SETUP ROUTING ===

// setupRoutes binds endpoints to fiber
func setupRoutes(ap *fiber.App) {
	ap.Use(logger.New())
	ap.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// basic handlers
	ap.Post("/api/login", handleLogin)
	ap.Post("/api/logout", handleLogout)

	api := ap.Group("/api", requireJWT)
	api.Get("/data", handleGetData)
	api.Get("/summary", handleGetSummary)
	api.Get("/prefixes", handleGetPrefixes)
	api.Get("/skip-log", handleGetSkipLogs)
	api.Get("/export-csv", handleExportCsv)

	ap.Static("/", "./web")
	ap.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})
}

// === HANDLER AUTENTIKASI ===

func handleLogin(c *fiber.Ctx) error {
	type Req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var rq Req
	if err := c.BodyParser(&rq); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if rq.Username != attendUser || rq.Password != attendPass {
		return c.Status(401).JSON(fiber.Map{"error": "username atau password salah"})
	}
	
	clm := jwtClaims{
		Username: rq.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
		},
	}
	tkn, err := jwt.NewWithClaims(jwt.SigningMethodHS256, clm).SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}
	
	c.Cookie(&fiber.Cookie{
		Name: "forming_token", Value: tkn, HTTPOnly: true, SameSite: "Lax", MaxAge: 43200,
	})
	return c.JSON(fiber.Map{"token": tkn, "username": rq.Username})
}

func handleLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{Name: "forming_token", Value: "", MaxAge: -1})
	return c.JSON(fiber.Map{"status": "ok"})
}

// === HANDLER DATA ===

func handleGetData(c *fiber.Ctx) error {
	rec, err := getRecords(
		c.Query("prefix"), c.Query("status"), c.Query("sort", "newest"), 
		c.Query("start_date"), c.Query("end_date"),
	)
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(rec)
}

func handleGetSummary(c *fiber.Ctx) error {
	smr, err := getSummary()
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(smr)
}

func handleGetPrefixes(c *fiber.Ctx) error {
	pfxs, err := lib.GetPrefixes(db)
	if err != nil { return c.Status(500).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(pfxs)
}

func handleGetSkipLogs(c *fiber.Ctx) error {
	skpLgs, err := lib.GetSkipLogs(db)
	if err != nil { return c.JSON([]map[string]interface{}{}) }
	return c.JSON(skpLgs)
}

func handleExportCsv(c *fiber.Ctx) error {
	mli, hnt := c.Query("start_date"), c.Query("end_date")
	sts, pfx := c.Query("status"), c.Query("prefix")

	pfxNm := "ALL"
	if pfx != "" && pfx != "all" { pfxNm = strings.ReplaceAll(pfx, " ", "_") }

	fnm := fmt.Sprintf("mdcw_export_%s_%s.csv", pfxNm, time.Now().Format("20060102"))
	if mli != "" && hnt != "" { fnm = fmt.Sprintf("mdcw_%s_%s_to_%s.csv", pfxNm, mli, hnt) }

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fnm))
	c.Set("Content-Type", "text/csv")

	rec, err := lib.GetRecordsByDateRange(db, mli, hnt, pfx, sts, "newest")
	if err != nil { return c.Status(500).SendString(err.Error()) }

	w := csv.NewWriter(c.Response().BodyWriter())
	w.Write([]string{"ID", "Timestamp", "Prefix", "Berat (g)", "Pack Count", "Status", "DataType", "Confidence"})
	
	for _, r := range rec {
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
