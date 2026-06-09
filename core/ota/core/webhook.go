package core

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func HandleEMQXWebhook(c *fiber.Ctx) error {
	if WebhookToken != "" && c.Get("X-Webhook-Token") != WebhookToken {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var psn EMQXWebhook
	if err := c.BodyParser(&psn); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse body"})
	}

	ndId := psn.ClientID
	if ndId == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty clientid"})
	}

	isTek := strings.HasPrefix(ndId, "web-") ||
		strings.HasPrefix(ndId, "megdev-") ||
		strings.HasPrefix(ndId, "servfi-") ||
		strings.HasPrefix(ndId, "forming-") ||
		strings.HasPrefix(ndId, "forwarder-")

	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	ada := false
	if _, ok := NodeStatus[ndId]; ok {
		ada = ok
	}

	if isTek && !ada {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "technical_client"})
	}

	if !ada {
		NodeStatus[ndId] = &NodeInfo{}
	}
	inf := NodeStatus[ndId]

	wkt := time.Now().Format("2006-01-02 15:04:05")
	inf.Updated = wkt

	switch psn.Event {
	case "client.connected":
		inf.Status = "online"
		Lg("Webhook: Node %s connected", ndId)
	case "client.disconnected":
		inf.Status = "offline"
		Lg("Webhook: Node %s disconnected (reason: %s)", ndId, psn.Reason)
	default:
		return c.JSON(fiber.Map{"status": "ignored", "event": psn.Event})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
