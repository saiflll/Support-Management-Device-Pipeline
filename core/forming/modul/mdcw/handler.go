package mdcw

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(api fiber.Router) {
	api.Get("/mdcw/data", handleGetData)
	api.Get("/mdcw/summary", handleGetSummary)
	api.Get("/mdcw/prefixes", handleGetPrefixes)
	api.Get("/mdcw/skip-log", handleGetSkipLogs)
	api.Get("/mdcw/export-csv", handleExportCsv)
	api.Get("/mdcw/daily-stats", handleGetDailyStats)
}

func handleGetData(c *fiber.Ctx) error {
	rec, err := GetRecords(
		c.Query("prefix"), c.Query("status"), c.Query("sort", "newest"),
		c.Query("start_date"), c.Query("end_date"),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rec)
}

func handleGetSummary(c *fiber.Ctx) error {
	smr, err := GetSummary()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(smr)
}

func handleGetPrefixes(c *fiber.Ctx) error {
	pfxs, err := GetPrefixes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pfxs)
}

func handleGetSkipLogs(c *fiber.Ctx) error {
	skpLgs, err := GetSkipLogs()
	if err != nil {
		return c.JSON([]map[string]interface{}{})
	}
	return c.JSON(skpLgs)
}

func handleExportCsv(c *fiber.Ctx) error {
	mli, hnt := c.Query("start_date"), c.Query("end_date")
	sts, pfx := c.Query("status"), c.Query("prefix")

	pfxNm := "ALL"
	if pfx != "" && pfx != "all" {
		pfxNm = strings.ReplaceAll(pfx, " ", "_")
	}

	fnm := fmt.Sprintf("mdcw_export_%s_%s.csv", pfxNm, time.Now().Format("20060102"))
	if mli != "" && hnt != "" {
		fnm = fmt.Sprintf("mdcw_%s_%s_to_%s.csv", pfxNm, mli, hnt)
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fnm))
	c.Set("Content-Type", "text/csv")

	rec, err := GetRecordsByDateRange(mli, hnt, pfx, sts, "newest")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	w := csv.NewWriter(c.Response().BodyWriter())
	w.Write([]string{"ID", "Timestamp", "Prefix", "Berat (g)", "Pack Count", "Status", "DataType", "Confidence"})

	for _, r := range rec {
		st := ""
		switch r.Reg5 {
		case 41, 521, 553:
			st = "OK"
		case 8:
			st = "MATI"
		case 9, 90:
			st = "IDLE"
		case 8201:
			st = "METAL"
		case 25:
			st = "UNDER"
		case 73:
			st = "OVER"
		default:
			st = fmt.Sprintf("UNKNOWN (%d)", r.Reg5)
		}
		w.Write([]string{
			fmt.Sprintf("%d", r.ID), r.Ts, r.Prefix, r.WeightFormatted, fmt.Sprintf("%d", r.Reg2), st,
			r.DataType, fmt.Sprintf("%.2f", r.Confidence),
		})
	}
	w.Flush()
	return nil
}

func handleGetDailyStats(c *fiber.Ctx) error {
	days := 7
	if d := c.Query("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}
	stats, err := GetDailyStats(days)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}
