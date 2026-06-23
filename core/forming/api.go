package main

import (
	"forming/modul/mdcw"
	"forming/modul/sp"
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
	tkn := c.Get("Authorization")
	if len(tkn) > 7 && strings.HasPrefix(tkn, "Bearer ") {
		tkn = tkn[7:]
	}
	if tkn == "" {
		tkn = c.Cookies("forming_token")
	}
	if tkn == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	prs, err := jwt.ParseWithClaims(tkn, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !prs.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}
	return c.Next()
}

func setupRoutes(ap *fiber.App) {
	ap.Use(logger.New())
	ap.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	ap.Post("/api/login", handleLogin)
	ap.Post("/api/logout", handleLogout)

	api := ap.Group("/api", requireJWT)

	mdcw.SetupRoutes(api)
	sp.SetupRoutes(api)

	ap.Static("/", "./web")
	ap.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})
}

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
