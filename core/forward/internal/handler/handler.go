package handler

import (
	"IoTT/internal/config"
	"IoTT/internal/headroom"
	"IoTT/internal/models"
	"IoTT/internal/processor"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

func HandleCompress(c *fiber.Ctx) error {
	if config.HeadroomClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Headroom service is not configured",
		})
	}

	var req headroom.CompressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON payload",
		})
	}

	resp, err := config.HeadroomClient.Compress(req)
	if err != nil {
		log.Printf("Headroom compression failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Compression failed: %v", err),
		})
	}

	return c.JSON(resp)
}


func HandleSensorData(c *fiber.Ctx) error {
	var dt []models.AreaData
	if err := c.BodyParser(&dt); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON payload. Expected an array of area data objects.",
		})
	}

	if len(dt) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Received empty data list."})
	}

	// Langsung panggil ProcessSensorData tanpa transaksi
	jml, err := processor.ProcessSensorData(dt)
	if err != nil {
		// Jika ada error saat parsing atau validasi awal, kembalikan error
		// Perhatikan bahwa error dari database (seperti koneksi) akan ditangani oleh worker
		// dan di-log secara terpisah, tidak menghentikan flow HTTP ini.
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"status":  "accepted with errors",
			"message": "Sebagian data mungkin tidak valid, namun data yang valid telah diterima untuk diproses.",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("Successfully accepted %d sensor readings for processing.", jml),
	})
}

func HandleTelegramWebhook(c *fiber.Ctx) error {
	var update interface{}

	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "invalid update format",
		})
	}

	log.Printf("Received Telegram update: %+v", update)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}
