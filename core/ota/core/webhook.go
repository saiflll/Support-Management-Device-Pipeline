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

	var payload EMQXWebhook
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse body"})
	}

	nodeID := payload.ClientID
	if nodeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty clientid"})
	}

	isTechnical := strings.HasPrefix(nodeID, "web-") ||
		strings.HasPrefix(nodeID, "megdev-") ||
		strings.HasPrefix(nodeID, "servfi-") ||
		strings.HasPrefix(nodeID, "forming-") ||
		strings.HasPrefix(nodeID, "forwarder-")

	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	exists := false
	if _, ok := NodeStatus[nodeID]; ok {
		exists = ok
	}

	if isTechnical && !exists {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "technical_client"})
	}

	if !exists {
		NodeStatus[nodeID] = &NodeInfo{}
	}
	info := NodeStatus[nodeID]

	now := time.Now().Format("2006-01-02 15:04:05")
	info.Updated = now

	switch payload.Event {
	case "client.connected":
		info.Status = "online"
		Lg("Webhook: Node %s connected", nodeID)
	case "client.disconnected":
		info.Status = "offline"
		Lg("Webhook: Node %s disconnected (reason: %s)", nodeID, payload.Reason)
	default:
		return c.JSON(fiber.Map{"status": "ignored", "event": payload.Event})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
