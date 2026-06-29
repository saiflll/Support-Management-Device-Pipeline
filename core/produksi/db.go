package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	dbName string
)

// initDB establishes connection to local PostgreSQL database
func initDB() {
	hst := getEnv("DB_HOST", "localhost")
	prt := getEnv("DB_PORT", "5432")
	usr := getEnv("DB_USER", "postgres")
	pwd := getEnv("DB_PASSWORD", "") // Jangan hardcode password — set via env
	dbName = getEnv("DB_NAME", "servfi")

	if pwd == "" {
		lg("⚠️ DB_PASSWORD tidak diset, koneksi mungkin gagal.")
	}

	cnn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", usr, pwd, hst, prt, dbName)

	var err error
	db, err = sql.Open("postgres", cnn)
	if err != nil {
		hndlErr("initDB.Open", err)
		return
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err = db.Ping(); err != nil {
		hndlErr("initDB.Ping", err)
	} else {
		lg("✅ Connected to PostgreSQL database at %s:%s / db: %s", hst, prt, dbName)
	}
}

func closeDB() {
	if db != nil {
		db.Close()
		lg("🔌 Database connection closed")
	}
}

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

// ── Columns list (single source of truth) ──────────────────────────
var dataCols = []string{
	"id_record", "factory", "kode_produk", "qty_per_pack", "pack_per_karton",
	"gramasi_pack", "gramasi_karton", "tanggal_produksi", "shift",
	"tanggal_best_before", "kode_ketentuan", "kode_batch", "tanggal_record",
}

const selectCols = `id_record, factory, kode_produk, qty_per_pack, pack_per_karton,
	gramasi_pack, gramasi_karton, tanggal_produksi, shift,
	tanggal_best_before, kode_ketentuan, kode_batch, tanggal_record`

// getLatestData retrieves the latest 100 logs ordered by newest first.
func getLatestData() ([]map[string]interface{}, []string, error) {
	query := fmt.Sprintf(`SELECT %s FROM conveyor_logs ORDER BY id_record DESC LIMIT 100`, selectCols)
	data, err := queryPostgresToMap(query)
	return data, dataCols, err
}

// getDataByRange retrieves logs within a date range from PostgreSQL.
func getDataByRange(startDate, endDate string) ([]map[string]interface{}, []string, error) {
	startStr := startDate + " 00:00:00"
	endStr := endDate + " 23:59:59"
	query := fmt.Sprintf(`
		SELECT %s FROM conveyor_logs
		WHERE tanggal_record BETWEEN $1 AND $2
		ORDER BY id_record DESC LIMIT 1000`, selectCols)
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
	today := time.Now().Format("2006-01-02")
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
		var shift string
		var cnt int
		if err := rows.Scan(&shift, &cnt); err != nil {
			continue
		}
		summary.PerShift[shift] = cnt
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
		if err := rows.Scan(&s.Shift, &s.Hour, &s.Count); err != nil {
			continue
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
