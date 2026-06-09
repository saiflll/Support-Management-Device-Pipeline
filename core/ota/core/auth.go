package core

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth(c *fiber.Ctx) error {
	ss, err := Store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	if ss.Get("authenticated") != true {
		return c.Redirect("/login")
	}

	return c.Next()
}

func HandleShowLogin(c *fiber.Ctx) error {
	return RenderLogin(c, nil)
}

func HandleLogin(c *fiber.Ctx) error {
	ss, err := Store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}

	kdSmt := strings.ToLower(c.FormValue("code"))
	if kdSmt == "" {
		return c.Render("login", fiber.Map{"error": "Kode tidak boleh kosong."})
	}

	kdVal := ss.Get("auth_code")
	expVal := ss.Get("auth_expires")

	if kdVal == nil || expVal == nil {
		return c.Render("login", fiber.Map{"error": "Sesi tidak ditemukan. Silakan minta kode baru."})
	}

	kd, ok1 := kdVal.(string)

	var expUnix int64
	var ok2 bool
	switch v := expVal.(type) {
	case int64:
		expUnix = v
		ok2 = true
	case int:
		expUnix = int64(v)
		ok2 = true
	case float64:
		expUnix = int64(v)
		ok2 = true
	}

	if !ok1 || !ok2 {
		return c.Render("login", fiber.Map{"error": "Data sesi korup. Silakan minta kode baru."})
	}

	expWkt := time.Unix(expUnix, 0)

	if kd != kdSmt || time.Now().After(expWkt) {
		return RenderLogin(c, fiber.Map{"error": "Kode verifikasi salah atau sudah kadaluarsa."})
	}

	ss.Delete("auth_code")
	ss.Delete("auth_expires")
	ss.Set("authenticated", true)
	if err := ss.Save(); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Gagal menyimpan sesi")
	}

	return c.Redirect("/")
}

func RenderLogin(c *fiber.Ctx, dt fiber.Map) error {
	c.Type("html")
	tmp, err := template.New("login").Parse(loginHTML)
	if err != nil {
		return c.Status(500).SendString("Template error: " + err.Error())
	}
	var buf bytes.Buffer
	if err := tmp.Execute(&buf, dt); err != nil {
		return c.Status(500).SendString("Execute error: " + err.Error())
	}
	return c.Send(buf.Bytes())
}

func HandleLogout(c *fiber.Ctx) error {
	ss, err := Store.Get(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Session error")
	}
	ss.Destroy()
	return c.Redirect("/login")
}

func HandleRequestCode(c *fiber.Ctx) error {
	if TelegramBotToken == "" || TelegramChatID == "" {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Layanan Telegram tidak dikonfigurasi di server.",
		})
	}

	ss, err := Store.Get(c)
	if err != nil {
		HndlErr("Error getting session", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Gagal membuat sesi."})
	}

	b := make([]byte, 3)
	rand.Read(b)
	kd := hex.EncodeToString(b)

	ss.Set("auth_code", kd)
	ss.Set("auth_expires", time.Now().Add(5*time.Minute).Unix())

	if err := ss.Save(); err != nil {
		HndlErr("Error saving session", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": fmt.Sprintf("Gagal menyimpan sesi: %v", err)})
	}

	Lg("Code generated and saved: %s", kd)

	psn := fmt.Sprintf("```json\n{\n  \"event\": \"AUTH_CODE_GENERATED\",\n  \"service\": \"OTA_CORE\",\n  \"auth_code\": \"%s\",\n  \"expires\": \"5m\",\n  \"status\": \"pending\"\n}\n```", kd)
	go SendTelegramMessage(psn)

	return c.JSON(fiber.Map{"status": "ok"})
}

const loginHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login | OTA Core Admin</title>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary: #6366f1;
            --primary-hover: #4f46e5;
            --bg: #0f172a;
            --card-bg: rgba(30, 41, 59, 0.7);
            --text: #f8fafc;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Outfit', sans-serif;
            background: radial-gradient(circle at top left, #1e1b4b, #0f172a);
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
            color: var(--text);
            overflow: hidden;
        }
        .container {
            background: var(--card-bg);
            backdrop-filter: blur(12px);
            padding: 2.5rem;
            border-radius: 24px;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
            width: 100%;
            max-width: 400px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            animation: fadeIn 0.6s ease-out;
        }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }
        h1 { font-size: 1.875rem; font-weight: 600; margin-bottom: 0.5rem; text-align: center; }
        p.subtitle { color: #94a3b8; text-align: center; margin-bottom: 2rem; font-size: 0.875rem; }
        .form-group { margin-bottom: 1.5rem; }
        label { display: block; margin-bottom: 0.5rem; font-size: 0.875rem; color: #cbd5e1; }
        input {
            width: 100%;
            padding: 0.75rem 1rem;
            background: rgba(15, 23, 42, 0.6);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 12px;
            color: white;
            font-size: 1rem;
            transition: all 0.3s;
        }
        input:focus { outline: none; border-color: var(--primary); box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2); }
        .btn {
            width: 100%;
            padding: 0.75rem;
            border: none;
            border-radius: 12px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            margin-bottom: 1rem;
            display: block;
            text-align: center;
            text-decoration: none;
        }
        .btn-primary { background: var(--primary); color: white; }
        .btn-primary:hover { background: var(--primary-hover); transform: translateY(-2px); }
        .btn-secondary { background: rgba(255, 255, 255, 0.05); color: #e2e8f0; border: 1px solid rgba(255, 255, 255, 0.1); }
        .btn-secondary:hover { background: rgba(255, 255, 255, 0.1); }
        .error { background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.2); color: #fca5a5; padding: 0.75rem; border-radius: 12px; margin-bottom: 1.5rem; font-size: 0.875rem; text-align: center; }
        .success { background: rgba(34, 197, 94, 0.1); border: 1px solid rgba(34, 197, 94, 0.2); color: #86efac; padding: 0.75rem; border-radius: 12px; margin-bottom: 1.5rem; font-size: 0.875rem; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <h1>OTA Core Admin</h1>
        <p class="subtitle">Enter authentication code from Telegram.</p>
        
        {{if .error}}
        <div class="error">{{.error}}</div>
        {{end}}

        <form action="/login" method="POST">
            <div class="form-group">
                <label for="code">Verification Code</label>
                <input type="text" id="code" name="code" placeholder="e.g. a1b2c3" required autocomplete="off">
            </div>
            <button type="submit" class="btn btn-primary">Login Now</button>
        </form>
        
        <button type="button" class="btn btn-secondary" id="requestBtn">Request New Code</button>
    </div>

    <script>
        document.getElementById('requestBtn').onclick = async function() {
            const btn = this;
            const originalText = btn.innerText;
            btn.innerText = 'Sending...';
            btn.disabled = true;

            try {
                const resp = await fetch('/request-code', { method: 'POST' });
                const result = await resp.json();
                if (result.status === 'ok') {
                    alert('New code sent to Telegram!');
                } else {
                    alert('Error: ' + (result.message || 'Unknown error'));
                }
            } catch (e) {
                alert('Connection error.');
            } finally {
                btn.innerText = originalText;
                btn.disabled = false;
            }
        };
    </script>
</body>
</html>`
