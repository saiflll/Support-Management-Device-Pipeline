package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	externalDB        *sql.DB
	lastProcessedID   int64
	lastProcessedLock sync.RWMutex
	extDBName         string
)

// ProductMaster matches the structure of products in the master screenshot
type ProductMaster struct {
	Kode     string
	Nama     string
	QtyPack  int
	LifeSpan int
	Gram     int
}

// BarcodeData holds the parsed components of the conveyor barcode
type BarcodeData struct {
	Factory         string
	KodeProduk      string
	QtyPerPack      int
	TanggalProduksi string
	Shift           int
	BestBefore      string
	KodeKetentuan   string
	KodeBatch       string
}

// StartScraper initializes PostgreSQL tables, MySQL connection, and starts scraper loop
func StartScraper() {
	lg("[SCRAPER] Inisialisasi background scraper...")

	// 1. Setup tabel lokal PostgreSQL (menggunakan db global dari main.go)
	setupLocalConveyorTables()

	// 2. Load cached ID
	loadLastProcessedID()

	// 3. Connect ke MySQL luar
	initExternalDB()

	lg("[SCRAPER] Scraper worker aktif berjalan...")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runScrapeCycle()
	}
}

// initExternalDB establishes connection to the external MySQL database
func initExternalDB() {
	hst := getEnv("PROD_DB_HOST", "10.201.40.2")
	prt := getEnv("PROD_DB_PORT", "3306")
	usr := getEnv("PROD_DB_USER", "ppa")
	pwd := getEnv("PROD_DB_PASSWORD", "PestaPora123")
	extDBName = getEnv("PROD_DB_NAME", "db_konveyor")

	cnn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", usr, pwd, hst, prt, extDBName)

	var err error
	externalDB, err = sql.Open("mysql", cnn)
	if err != nil {
		log.Printf("[SCRAPER] ERROR mysql open connection: %v", err)
		return
	}

	externalDB.SetConnMaxLifetime(time.Minute * 3)
	externalDB.SetMaxOpenConns(5)
	externalDB.SetMaxIdleConns(5)

	if err = externalDB.Ping(); err != nil {
		log.Printf("[SCRAPER] ERROR mysql ping: %v", err)
	} else {
		lg("[SCRAPER] ✅ Terhubung ke MySQL luar di %s:%s", hst, prt)
	}
}

// setupLocalConveyorTables creates conveyor-related tables in local PostgreSQL and seeds product master
func setupLocalConveyorTables() {
	if db == nil {
		log.Println("[SCRAPER] ERROR: local PostgreSQL 'db' is nil")
		return
	}

	// Create master_produk table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS master_produk (
			kode VARCHAR(50) PRIMARY KEY,
			nama VARCHAR(255) NOT NULL,
			qty_pack INT NOT NULL,
			lifespan INT NOT NULL,
			gram INT NOT NULL
		);
	`)
	if err != nil {
		log.Printf("[SCRAPER] Error creating master_produk: %v", err)
	}

	// Create conveyor_logs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS conveyor_logs (
			id_record BIGINT PRIMARY KEY,
			factory VARCHAR(50),
			kode_produk VARCHAR(50),
			qty_per_pack INT,
			tanggal_produksi VARCHAR(20),
			shift INT,
			tanggal_best_before VARCHAR(20),
			kode_ketentuan VARCHAR(10),
			kode_batch VARCHAR(10),
			tanggal_record TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("[SCRAPER] Error creating conveyor_logs: %v", err)
	}

	// Seed master_produk if empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM master_produk").Scan(&count)
	if err == nil && count == 0 {
		lg("[SCRAPER] 🌱 Seeding master_produk table in PostgreSQL...")
		seeds := []ProductMaster{
			{"100209", "ADONAN PANGSIT", 10, 4, 1500},
			{"100211", "AYAM CINCANG", 12, 2, 1000},
			{"100238", "KERUPUK MIE", 6, 2, 1035},
			{"100239", "KULIT PANGSIT", 21, 2, 1035},
			{"100244", "LUMPIA UDANG", 38, 2, 300},
			{"100245", "MIE", 14, 2, 1020},
			{"100256", "SIOMAY DIMSUM", 18, 4, 720},
			{"100286", "UDANG RAMBUTAN (PEN)", 20, 4, 600},
			{"100294", "UDANG KEJU", 12, 4, 920},
			{"100339", "CABAI FROZEN", 9, 6, 1005},
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("[SCRAPER] Error beginning seed transaction: %v", err)
			return
		}

		for _, s := range seeds {
			_, err = tx.Exec(`
				INSERT INTO master_produk (kode, nama, qty_pack, lifespan, gram) 
				VALUES ($1, $2, $3, $4, $5) 
				ON CONFLICT (kode) DO NOTHING`,
				s.Kode, s.Nama, s.QtyPack, s.LifeSpan, s.Gram)
			if err != nil {
				tx.Rollback()
				log.Printf("[SCRAPER] Error executing seed: %v", err)
				return
			}
		}
		tx.Commit()
		lg("[SCRAPER] 🌱 Seeding completed successfully!")
	}
}

// loadLastProcessedID initializes the in-memory cache from PostgreSQL
func loadLastProcessedID() {
	if db == nil {
		return
	}

	var maxID sql.NullInt64
	err := db.QueryRow("SELECT MAX(id_record) FROM conveyor_logs").Scan(&maxID)
	if err != nil {
		log.Printf("[SCRAPER] Error loading max id: %v", err)
		return
	}

	lastProcessedLock.Lock()
	if maxID.Valid {
		lastProcessedID = maxID.Int64
	} else {
		lastProcessedID = 0
	}
	lastProcessedLock.Unlock()

	lg("[SCRAPER] 💾 Cache inisialisasi: lastProcessedID = %d", lastProcessedID)
}

// getMasterMap reads the lookup table from local PostgreSQL database
func getMasterMap() (map[string]ProductMaster, error) {
	if db == nil {
		return nil, fmt.Errorf("local database is nil")
	}

	rows, err := db.Query("SELECT kode, nama, qty_pack, lifespan, gram FROM master_produk")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	masters := make(map[string]ProductMaster)
	for rows.Next() {
		var p ProductMaster
		if err := rows.Scan(&p.Kode, &p.Nama, &p.QtyPack, &p.LifeSpan, &p.Gram); err != nil {
			return nil, err
		}
		masters[p.Kode] = p
	}
	return masters, nil
}

// parseBarcode extracts components from conveyor barcode
func parseBarcode(barcode string) BarcodeData {
	parts := strings.Split(barcode, "-")
	if len(parts) < 6 {
		return BarcodeData{}
	}

	// Part 0: F3/100294
	p0 := parts[0]
	p0Parts := strings.Split(p0, "/")
	var factory, kodeProduk string
	if len(p0Parts) == 2 {
		factory = p0Parts[0]
		kodeProduk = p0Parts[1]
	} else {
		kodeProduk = p0
	}

	// Part 1: Qty
	qty, _ := strconv.Atoi(parts[1])

	// Part 2: Tanggal Produksi
	tglProduksi := parts[2]

	// Part 3: Shift
	shift, _ := strconv.Atoi(parts[3])

	// Part 4: Best Before
	bestBefore := parts[4]

	// Part 5: N(02)
	p5 := parts[5]
	var kodeKetentuan, kodeBatch string
	if idxOpen := strings.Index(p5, "("); idxOpen != -1 {
		kodeKetentuan = p5[:idxOpen]
		if idxClose := strings.Index(p5, ")"); idxClose != -1 && idxClose > idxOpen {
			kodeBatch = p5[idxOpen+1 : idxClose]
		} else {
			kodeBatch = p5[idxOpen:]
		}
	} else {
		kodeKetentuan = p5
	}

	return BarcodeData{
		Factory:         factory,
		KodeProduk:      kodeProduk,
		QtyPerPack:      qty,
		TanggalProduksi: tglProduksi,
		Shift:           shift,
		BestBefore:      bestBefore,
		KodeKetentuan:   kodeKetentuan,
		KodeBatch:       kodeBatch,
	}
}

// runScrapeCycle executes a single cycle of checking and fetching data
func runScrapeCycle() {
	if externalDB == nil {
		initExternalDB()
		if externalDB == nil {
			return
		}
	}

	// Read lastProcessedID cache
	lastProcessedLock.RLock()
	currentLastID := lastProcessedID
	lastProcessedLock.RUnlock()

	// 1. Pengecekan ringan MAX ID ke MySQL luar
	var maxMySQLID sql.NullInt64
	err := externalDB.QueryRow("SELECT MAX(ID_RecordProduk) FROM `record_produk`").Scan(&maxMySQLID)
	if err != nil {
		log.Printf("[SCRAPER] Error querying max mysql id: %v", err)
		if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "refused") {
			externalDB = nil
		}
		return
	}

	// Jika tidak ada data baru, lewati
	if !maxMySQLID.Valid || maxMySQLID.Int64 <= currentLastID {
		return
	}

	// 2. Tarik 10 data terbaru
	rows, err := externalDB.Query(`
		SELECT ID_RecordProduk, BarcodeProduk, JamRecordProduk 
		FROM record_produk 
		WHERE ID_RecordProduk > ? 
		ORDER BY ID_RecordProduk ASC 
		LIMIT 10
	`, currentLastID)
	if err != nil {
		log.Printf("[SCRAPER] Error fetching batch data: %v", err)
		return
	}
	defer rows.Close()

	var batch []map[string]interface{}
	for rows.Next() {
		var id int64
		var barcode string
		var jamRecord time.Time

		if err := rows.Scan(&id, &barcode, &jamRecord); err != nil {
			log.Printf("[SCRAPER] Error scanning mysql row: %v", err)
			return
		}

		batch = append(batch, map[string]interface{}{
			"id_record":      id,
			"barcode":        barcode,
			"tanggal_record": jamRecord,
		})
	}

	if len(batch) == 0 {
		return
	}

	// 3. Proses dan simpan dalam transaksi (batch insert)

	tx, err := db.Begin()
	if err != nil {
		log.Printf("[SCRAPER] Error starting postgres transaction: %v", err)
		return
	}

	var latestProcessedID int64
	for _, item := range batch {
		idRecord := item["id_record"].(int64)
		barcodeStr := item["barcode"].(string)
		tglRecord := item["tanggal_record"].(time.Time)

		bc := parseBarcode(barcodeStr)

		// Simpan dengan ON CONFLICT DO NOTHING
		_, err = tx.Exec(`
			INSERT INTO conveyor_logs (
				id_record, factory, kode_produk, qty_per_pack, 
				tanggal_produksi, shift, 
				tanggal_best_before, kode_ketentuan, kode_batch, tanggal_record
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id_record) DO NOTHING
		`, idRecord, bc.Factory, bc.KodeProduk, bc.QtyPerPack,
			bc.TanggalProduksi, bc.Shift,
			bc.BestBefore, bc.KodeKetentuan, bc.KodeBatch, tglRecord)

		if err != nil {
			tx.Rollback()
			log.Printf("[SCRAPER] Error executing insert in postgres: %v", err)
			return
		}

		latestProcessedID = idRecord
	}

	if err = tx.Commit(); err != nil {
		log.Printf("[SCRAPER] Error committing transaction in postgres: %v", err)
		return
	}

	// 4. Update memory cache
	lastProcessedLock.Lock()
	if latestProcessedID > lastProcessedID {
		lastProcessedID = latestProcessedID
	}
	lastProcessedLock.Unlock()

	lg("[SCRAPER] 💾 Berhasil mensinkronkan batch data ke local. lastProcessedID baru = %d", latestProcessedID)
}
