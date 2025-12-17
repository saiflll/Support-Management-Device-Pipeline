package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"
)

var Store *session.Store

// InitAuth initializes the session store
func InitAuth() {
	Store = session.New(session.Config{
		Storage:        memory.New(),
		Expiration:     24 * time.Hour,
		KeyLookup:      "cookie:session_id",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		KeyGenerator: func() string {
			b := make([]byte, 16)
			rand.Read(b)
			return hex.EncodeToString(b)
		},
	})
}

// RequireAuth middleware checks if user is authenticated
func RequireAuth(c *fiber.Ctx) error {
	if Store == nil {
		InitAuth()
	}

	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	if sess.Get("authenticated") != true {
		// Redirect to OTA login
		return c.Redirect("/login")
	}

	return c.Next()
}
