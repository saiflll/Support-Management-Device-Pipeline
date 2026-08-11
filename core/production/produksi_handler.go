package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ── Columns list (single source of truth) ──────────────────────────
var dataCols = []string{
	"id_record", "factory", "kode_produk", "qty_per_pack", "pack_per_karton",
	"gramasi_pack", "gramasi_karton", "tanggal_produksi", "shift",
	"tanggal_best_before", "kode_ketentuan", "kode_batch", "tanggal_record",
}

const selectCols = `id_record, factory, kode_produk, qty_per_pack, pack_per_karton,
	gramasi_pack, gramasi_karton, tanggal_produksi, shift,
	tanggal_best_before, kode_ketentuan, kode_batch, tanggal_record`

// queryPostgresToMap queries PostgreSQL and returns rows as a slice of maps.
func queryPostgresToMap(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is not open")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			if val == nil {
				m[colName] = nil
			} else if b, ok := val.([]byte); ok {
				m[colName] = string(b)
			} else if t, ok := val.(time.Time); ok {
				m[colName] = t.Format("2006-01-02 15:04:05")
			} else {
				m[colName] = val
			}
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// timeNow helper
func timeNow() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("2006-01-02")
}

// getLatestData retrieves the latest 100 logs ordered by newest first.
func getLatestData() ([]map[string]interface{}, []string, error) {
	query := `
		SELECT c.id_record, c.factory, c.kode_produk, c.qty_per_pack, 
		       COALESCE(m.qty_pack, 0) as pack_per_karton, 
		       COALESCE(CAST(m.gram AS NUMERIC(10,2)) / NULLIF(m.qty_pack, 0), 0) as gramasi_pack, 
		       COALESCE(m.gram, 0) as gramasi_karton, 
		       c.tanggal_produksi, c.shift, 
		       c.tanggal_best_before, c.kode_ketentuan, c.kode_batch, c.tanggal_record 
		FROM conveyor_logs c
		LEFT JOIN master_produk m ON c.kode_produk = m.kode
		ORDER BY c.id_record DESC LIMIT 100`
	data, err := queryPostgresToMap(query)
	return data, dataCols, err
}

// getDataByRange retrieves logs within a date range from PostgreSQL.
func getDataByRange(startDate, endDate string) ([]map[string]interface{}, []string, error) {
	startStr := startDate + " 00:00:00"
	endStr := endDate + " 23:59:59"
	query := `
		SELECT c.id_record, c.factory, c.kode_produk, c.qty_per_pack, 
		       COALESCE(m.qty_pack, 0) as pack_per_karton, 
		       COALESCE(CAST(m.gram AS NUMERIC(10,2)) / NULLIF(m.qty_pack, 0), 0) as gramasi_pack, 
		       COALESCE(m.gram, 0) as gramasi_karton, 
		       c.tanggal_produksi, c.shift, 
		       c.tanggal_best_before, c.kode_ketentuan, c.kode_batch, c.tanggal_record 
		FROM conveyor_logs c
		LEFT JOIN master_produk m ON c.kode_produk = m.kode
		WHERE c.tanggal_record BETWEEN $1 AND $2
		ORDER BY c.id_record DESC LIMIT 1000`
	data, err := queryPostgresToMap(query, startStr, endStr)
	return data, dataCols, err
}

// ─── Analytics Queries ────────────────────────────────────────────

// SummaryToday mengembalikan total record hari ini dan breakdown per shift.
type SummaryToday struct {
	Total      int            `json:"total"`
	PerShift   map[string]int `json:"per_shift"`
	LastUpdate string         `json:"last_update"`
}

func getSummaryToday() (SummaryToday, error) {
	today := timeNow()
	startStr := today + " 00:00:00"
	endStr := today + " 23:59:59"

	rows, err := db.Query(`
		SELECT shift, COUNT(*) as cnt
		FROM conveyor_logs
		WHERE tanggal_record BETWEEN $1 AND $2
		GROUP BY shift ORDER BY shift`, startStr, endStr)
	if err != nil {
		return SummaryToday{}, err
	}
	defer rows.Close()

	summary := SummaryToday{PerShift: make(map[string]int)}
	for rows.Next() {
		var shift int
		var cnt int
		if err := rows.Scan(&shift, &cnt); err != nil {
			continue
		}
		shiftStr := fmt.Sprintf("%d", shift)
		summary.PerShift[shiftStr] = cnt
		summary.Total += cnt
	}
	summary.LastUpdate = time.Now().Format("15:04:05")
	return summary, rows.Err()
}

// DailyCount untuk bar chart produksi mingguan.
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func getWeeklyProduction() ([]DailyCount, error) {
	rows, err := db.Query(`
		SELECT DATE(tanggal_record) as hari, COUNT(*) as cnt
		FROM conveyor_logs
		WHERE tanggal_record >= NOW() - INTERVAL '7 days'
		GROUP BY hari
		ORDER BY hari ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyCount
	for rows.Next() {
		var d DailyCount
		var t time.Time
		if err := rows.Scan(&t, &d.Count); err != nil {
			continue
		}
		d.Date = t.Format("2006-01-02")
		result = append(result, d)
	}
	return result, rows.Err()
}

// ProductDist untuk pie chart distribusi kode produk.
type ProductDist struct {
	KodeProduk string `json:"kode_produk"`
	Count      int    `json:"count"`
}

func getProductDistribution() ([]ProductDist, error) {
	rows, err := db.Query(`
		SELECT kode_produk, COUNT(*) as cnt
		FROM conveyor_logs
		WHERE tanggal_record >= NOW() - INTERVAL '30 days'
		GROUP BY kode_produk
		ORDER BY cnt DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProductDist
	for rows.Next() {
		var d ProductDist
		if err := rows.Scan(&d.KodeProduk, &d.Count); err != nil {
			continue
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// ShiftTimeline untuk timeline produksi per shift dalam sehari.
type ShiftPoint struct {
	Shift string `json:"shift"`
	Hour  int    `json:"hour"`
	Count int    `json:"count"`
}

func getShiftTimeline(date string) ([]ShiftPoint, error) {
	startStr := date + " 00:00:00"
	endStr := date + " 23:59:59"
	rows, err := db.Query(`
		SELECT shift, EXTRACT(HOUR FROM tanggal_record::timestamp)::int as jam, COUNT(*) as cnt
		FROM conveyor_logs
		WHERE tanggal_record BETWEEN $1 AND $2
		GROUP BY shift, jam
		ORDER BY shift, jam`, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ShiftPoint
	for rows.Next() {
		var s ShiftPoint
		var shift int
		if err := rows.Scan(&shift, &s.Hour, &s.Count); err != nil {
			continue
		}
		s.Shift = fmt.Sprintf("%d", shift)
		result = append(result, s)
	}
	return result, rows.Err()
}

// ── Fiber Handlers ───────────────────────────────────────────────────

func handleDBStatus(c *fiber.Ctx) error {
	status := "connected"
	var details string
	if db == nil {
		status = "disconnected"
		details = "Database connection not initialized"
	} else if err := db.Ping(); err != nil {
		status = "error"
		details = err.Error()
	}
	return c.JSON(fiber.Map{"status": status, "details": details})
}

func handleLatest(c *fiber.Ctx) error {
	data, cols, err := getLatestData()
	if err != nil {
		hndlErr("handleLatest", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal mengambil data terbaru", "details": err.Error()})
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"columns": cols, "data": data})
}

func handleRange(c *fiber.Ctx) error {
	start := c.Query("start")
	end := c.Query("end")
	if start == "" || end == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Parameter 'start' dan 'end' wajib diisi (format YYYY-MM-DD)"})
	}
	data, cols, err := getDataByRange(start, end)
	if err != nil {
		hndlErr("handleRange", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal mengambil data range", "details": err.Error()})
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"columns": cols, "data": data})
}

func handleSummaryToday(c *fiber.Ctx) error {
	summary, err := getSummaryToday()
	if err != nil {
		hndlErr("handleSummaryToday", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}

func handleWeeklyProduction(c *fiber.Ctx) error {
	data, err := getWeeklyProduction()
	if err != nil {
		hndlErr("handleWeeklyProduction", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func handleProductDistribution(c *fiber.Ctx) error {
	data, err := getProductDistribution()
	if err != nil {
		hndlErr("handleProductDistribution", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func handleShiftTimeline(c *fiber.Ctx) error {
	date := c.Query("date")
	if date == "" {
		date = timeNow()
	}
	data, err := getShiftTimeline(date)
	if err != nil {
		hndlErr("handleShiftTimeline", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}
