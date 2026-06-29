package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory/v2"
)

var Store *session.Store

// InitAuth menginisialisasi session store untuk autentikasi.
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

// IsDevMode mengembalikan true jika aplikasi berjalan dalam mode development.
// Set environment variable DEV_MODE=true untuk bypass autentikasi.
func IsDevMode() bool {
	return os.Getenv("DEV_MODE") == "true"
}

// RequireAuth middleware untuk memvalidasi sesi pengguna.
// Jika DEV_MODE=true, autentikasi dilewati sepenuhnya.
func RequireAuth(c *fiber.Ctx) error {
	if IsDevMode() {
		log.Println("⚠️ [DEV_MODE] Auth bypass aktif — nonaktifkan di production!")
		return c.Next()
	}

	if Store == nil {
		InitAuth()
	}

	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	if sess.Get("authenticated") != true {
		return c.Redirect("/login")
	}

	return c.Next()
}
