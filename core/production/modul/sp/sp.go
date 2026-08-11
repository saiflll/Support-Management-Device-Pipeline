package sp

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"production/redis"
)

var (
	db         *sql.DB
	mqttClient mqtt.Client
	cldFwd     *spCloudForwarder
)

type spCloudForwarder struct {
	client  mqtt.Client
	topic   string
	enabled bool
	mu      sync.Mutex
}

func Init(database *sql.DB, mqttCli mqtt.Client) {
	db = database
	mqttClient = mqttCli

	createTable()

	if mqttClient != nil {
		subscribe(mqttClient)
	}

	InitSpCloudForwarder()
}

func InitSpCloudForwarder() {
	uri := os.Getenv("CLOUD_MQTT_BROKER_URI")
	if uri == "" {
		log.Println("[SP-CloudFwd] CLOUD_MQTT_BROKER_URI not set. Cloud forwarding INACTIVE.")
		cldFwd = &spCloudForwarder{enabled: false}
		return
	}

	tpc := os.Getenv("CLOUD_MQTT_TOPIC_SP")
	if tpc == "" {
		tpc = os.Getenv("CLOUD_MQTT_TOPIC_FORMING")
		if tpc == "" {
			tpc = "prod/mdcw"
		}
	}

	usr := os.Getenv("CLOUD_MQTT_USERNAME")
	pwd := os.Getenv("CLOUD_MQTT_PASSWORD")

	opt := mqtt.NewClientOptions()
	opt.AddBroker(uri)
	opt.SetClientID(fmt.Sprintf("sp-cloud-fwd-%d", time.Now().UnixNano()))
	opt.SetAutoReconnect(true)
	opt.SetKeepAlive(60 * time.Second)
	opt.SetConnectTimeout(10 * time.Second)
	opt.SetProtocolVersion(4)

	if usr != "" {
		opt.SetUsername(usr)
		opt.SetPassword(pwd)
		log.Printf("[SP-CloudFwd] Using cloud MQTT credentials: %s", usr)
	}

	opt.OnConnect = func(cln mqtt.Client) {
		log.Printf("[SP-CloudFwd] Connected to Cloud MQTT Broker: %s", uri)
	}
	opt.OnConnectionLost = func(cln mqtt.Client, err error) {
		log.Printf("[SP-CloudFwd] Connection lost: %v", err)
	}

	cln := mqtt.NewClient(opt)
	if tkn := cln.Connect(); tkn.Wait() && tkn.Error() != nil {
		log.Printf("[SP-CloudFwd] Connect error: %v", tkn.Error())
		log.Println("[SP-CloudFwd] Cloud forwarding active — will retry on publish.")
	}

	cldFwd = &spCloudForwarder{
		client:  cln,
		topic:   tpc,
		enabled: true,
	}

	log.Printf("[SP-CloudFwd] Cloud forwarder active -> broker: %s, topic: %s", uri, tpc)
}

func createTable() {
	qry := `
	CREATE TABLE IF NOT EXISTS production_sp (
		id SERIAL PRIMARY KEY,
		session_id VARCHAR(100),
		data TEXT,
		ts VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(qry); err != nil {
		log.Printf("[SP] createTable error: %v", err)
	} else {
		log.Println("[SP] Table 'production_sp' ensured")
	}
}

func subscribe(cln mqtt.Client) {
	tpc := "SP_data"
	if tkn := cln.Subscribe(tpc, 1, messageHandler); tkn.Wait() && tkn.Error() != nil {
		log.Printf("[SP] Error subscribing to topic %s: %v", tpc, tkn.Error())
	} else {
		log.Printf("[SP] Subscribed to topic: %s", tpc)
	}
}

func messageHandler(cln mqtt.Client, psn mqtt.Message) {
	payload := string(psn.Payload())

	parts := strings.SplitN(payload, "&", 3)
	if len(parts) < 3 {
		log.Printf("[SP] Invalid payload format: %s", payload)
		return
	}

	sessionID := parts[0]
	data := parts[1]
	ts := parts[2]

	insertData(sessionID, data, ts)

	go ForwardSpToCloud(sessionID, data, ts)
}

var sessionToMachineID = map[string]int{
	"SP1": 31,
	"SP2": 32,
	"SP3": 33,
}

func currentShift() int {
	hr := time.Now().Hour()
	switch {
	case hr >= 7 && hr < 15:
		return 1
	case hr >= 15 && hr < 23:
		return 2
	default:
		return 3
	}
}

func ForwardSpToCloud(sessionID, productCode, tsRaw string) {
	if cldFwd == nil || !cldFwd.enabled {
		return
	}

	macId, ok := sessionToMachineID[sessionID]
	if !ok {
		log.Printf("[SP-CloudFwd] Session '%s' not found in machine_id mapping. Data not forwarded.", sessionID)
		return
	}

	// Ambil qty_pack dari Redis cache (fallback ke database)
	qtyPack := 1
	if cacheVal, ok := redis.GetCache(redis.MasterProductKey + ":" + productCode); ok {
		var mp struct{ QtyPack int }
		if err := json.Unmarshal([]byte(cacheVal), &mp); err == nil {
			qtyPack = mp.QtyPack
			log.Printf("[SP-CloudFwd] Cache hit untuk produk %s: qty_pack=%d", productCode, qtyPack)
		}
	} else if db != nil {
		err := db.QueryRow("SELECT qty_pack FROM master_produk WHERE kode = $1", productCode).Scan(&qtyPack)
		if err != nil {
			log.Printf("[SP-CloudFwd] Gagal ambil qty_pack untuk produk %s: %v. Menggunakan default qty_pack=1", productCode, err)
		} else {
			// Cache hasil query untuk 30 menit
			mp := struct{ QtyPack int }{qtyPack}
			if b, err := json.Marshal(mp); err == nil {
				redis.SetCache(redis.MasterProductKey+":"+productCode, string(b), redis.CacheTTL)
			}
		}
	}

	sft := currentShift()

	wkt := time.Now().Format(time.RFC3339)
	if tsRaw != "" {
		wkt = tsRaw
	}

	// Format: CSV,machine_id,shift,output,nice,reject,rjMtl,rjOvr,rjUnd,rjOth,pwr,eff,ts,product_code
	csv := fmt.Sprintf(
		"CSV,%d,%d,%d,%d,%d,%d,%d,%d,%d,%.3f,%.2f,%s,%s",
		macId,
		sft,
		qtyPack, // output = jumlah pack per karton
		qtyPack, // nice = jumlah pack per karton
		0,       // reject
		0,       // reject_metal
		0,       // reject_overweight
		0,       // reject_underweight
		0,       // reject_other
		0.0,     // power
		100.0,   // efficiency
		wkt,
		productCode,
	)

	cldFwd.mu.Lock()
	defer cldFwd.mu.Unlock()

	if !cldFwd.client.IsConnected() {
		log.Println("[SP-CloudFwd] Client not connected, reconnecting before publish...")
		tkn := cldFwd.client.Connect()
		tkn.Wait()
	}

	tkn := cldFwd.client.Publish(cldFwd.topic, 1, false, csv)
	tkn.Wait()
	if err := tkn.Error(); err != nil {
		log.Printf("[SP-CloudFwd] Publish error: %v", err)
	} else {
		log.Printf("[SP-CloudFwd] Forwarded to cloud [%s] machine=%d shift=%d session=%s",
			cldFwd.topic, macId, sft, sessionID)
	}
}

func insertData(sessionID, data, ts string) {
	if db == nil {
		return
	}

	qry := `INSERT INTO production_sp (session_id, data, ts) VALUES ($1, $2, $3)`
	if _, err := db.Exec(qry, sessionID, data, ts); err != nil {
		log.Printf("[SP] insertData error: %v", err)
	} else {
		log.Printf("[SP] Data inserted: session=%s data=%s ts=%s", sessionID, data, ts)
	}
}

func SetupRoutes(api fiber.Router) {
	api.Get("/sp/data", handleGetData)
	api.Get("/sp/summary", handleGetSummary)
	api.Get("/sp/sessions", handleGetSessions)
	api.Get("/sp/export-csv", handleExportCsv)
	api.Get("/sp/daily-stats", handleGetDailyStats)
	api.Get("/sp/comparison", handleGetComparison)
}

func handleGetData(c *fiber.Ctx) error {
	rec, err := getRecords(
		c.Query("session_id"),
		c.Query("start_date"),
		c.Query("end_date"),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rec)
}

func handleGetSummary(c *fiber.Ctx) error {
	smr, err := getSummary()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(smr)
}

func handleGetSessions(c *fiber.Ctx) error {
	sessions, err := getSessions()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sessions)
}

func handleExportCsv(c *fiber.Ctx) error {
	sessionID := c.Query("session_id")
	mli := c.Query("start_date")
	hnt := c.Query("end_date")

	fnm := "sp_export.csv"
	if mli != "" && hnt != "" {
		fnm = fmt.Sprintf("sp_export_%s_to_%s.csv", mli, hnt)
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fnm))
	c.Set("Content-Type", "text/csv")

	rec, err := getRecords(sessionID, mli, hnt)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	w := csv.NewWriter(c.Response().BodyWriter())
	w.Write([]string{"ID", "Session ID", "Data", "Timestamp"})

	for _, r := range rec {
		w.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.SessionID,
			r.Data,
			r.Ts,
		})
	}
	w.Flush()
	return nil
}

func getRecords(sessionFltr, mliWkt, hntWkt string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	qry := "SELECT id, session_id, data, ts, created_at FROM production_sp WHERE 1=1"
	var args []interface{}
	argId := 1

	if sessionFltr != "" && sessionFltr != "all" {
		qry += fmt.Sprintf(" AND session_id = $%d", argId)
		args = append(args, sessionFltr)
		argId++
	}

	if mliWkt != "" {
		qry += fmt.Sprintf(" AND DATE(created_at) >= $%d", argId)
		args = append(args, mliWkt)
		argId++
	}
	if hntWkt != "" {
		qry += fmt.Sprintf(" AND DATE(created_at) <= $%d", argId)
		args = append(args, hntWkt)
		argId++
	}

	qry += " ORDER BY created_at DESC LIMIT 2000"

	rows, err := db.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Record
	for rows.Next() {
		var r Record
		var sid, dt, ts sql.NullString

		if err := rows.Scan(&r.ID, &sid, &dt, &ts, &r.CreatedAt); err != nil {
			return nil, err
		}

		if sid.Valid {
			r.SessionID = sid.String
		}
		if dt.Valid {
			r.Data = dt.String
		}
		if ts.Valid {
			r.Ts = ts.String
		}

		recs = append(recs, r)
	}

	return recs, nil
}

func getSummary() ([]Summary, error) {
	if db == nil {
		return []Summary{}, nil
	}

	qry := `
		SELECT 
			session_id, 
			COUNT(*) as total_count,
			MAX(ts) as last_scan,
			MIN(ts) as first_scan
		FROM production_sp
		WHERE DATE(created_at) = CURRENT_DATE
		GROUP BY session_id
		ORDER BY session_id
	`
	rows, err := db.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var smrs []Summary
	for rows.Next() {
		var s Summary
		var sid, lastTs, firstTs sql.NullString
		var cnt sql.NullInt64

		if err := rows.Scan(&sid, &cnt, &lastTs, &firstTs); err != nil {
			continue
		}

		if sid.Valid {
			s.SessionID = sid.String
		}
		if cnt.Valid {
			s.TotalCount = int(cnt.Int64)
		}
		if lastTs.Valid {
			s.LastScan = lastTs.String
		}
		if firstTs.Valid {
			s.FirstScan = firstTs.String
		}

		smrs = append(smrs, s)
	}

	return smrs, nil
}

func getSessions() ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	qry := `SELECT DISTINCT session_id FROM production_sp WHERE session_id IS NOT NULL ORDER BY session_id`
	rows, err := db.Query(qry)
	if err != nil {
		return []string{}, nil
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		if s != "" {
			sessions = append(sessions, s)
		}
	}

	return sessions, nil
}

type DailyStat struct {
	Date       string `json:"date"`
	TotalCount int    `json:"total_count"`
}

func GetDailyStats(days int) ([]DailyStat, error) {
	if db == nil {
		return []DailyStat{}, nil
	}

	prm := fmt.Sprintf("%d", days)
	qry := `
		SELECT DATE(created_at) as dt, COUNT(*) as total
		FROM production_sp
		WHERE created_at >= CURRENT_DATE - $1::INTEGER * INTERVAL '1 day'
		GROUP BY DATE(created_at)
		ORDER BY dt ASC
	`
	rows, err := db.Query(qry, prm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		var dt sql.NullTime
		var total sql.NullInt64

		if err := rows.Scan(&dt, &total); err != nil {
			continue
		}

		if dt.Valid {
			s.Date = dt.Time.Format("2006-01-02")
		}
		if total.Valid {
			s.TotalCount = int(total.Int64)
		}

		stats = append(stats, s)
	}

	return stats, nil
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

func handleGetComparison(c *fiber.Ctx) error {
	comparison, err := GetComparison()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(comparison)
}

func GetComparison() ([]ComparisonRow, error) {
	if db == nil {
		return []ComparisonRow{}, nil
	}

	type Product struct {
		Name    string
		QtyPack int
	}
	prods := make(map[string]Product)
	rows, err := db.Query("SELECT kode, nama, qty_pack FROM master_produk")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k, n string
			var q int
			if err := rows.Scan(&k, &n, &q); err == nil {
				prods[k] = Product{Name: n, QtyPack: q}
			}
		}
	}

	prefixToProduct := map[string]string{
		"MDCW1 (UK)":            "100294",
		"MDCW2 (Siomay)":        "100256",
		"MDCW3 (Pentol)":        "100286",
		"MDCW4 (AP)":            "100209",
		"MDCW5 (ACIN)":          "100211",
		"MDCW6 (Lumpia)":        "100244",
		"MDCW7 (Kulit/Kerupuk)": "100239",
		"MDCW8 (Mie)":           "100245",
		"MDCW9 (Mie)":           "100245",
	}

	sessionToPrefix := map[string]string{
		"SP1": "MDCW1 (UK)",
		"SP2": "MDCW2 (Siomay)",
		"SP3": "MDCW3 (Pentol)",
	}

	type Key struct {
		Date        string
		Prefix      string
		ProductCode string
	}

	mdcwData := make(map[Key]int)
	rowsMdcw, err := db.Query(`
		SELECT TO_CHAR(created_at, 'YYYY-MM-DD') as dt, prefix, COUNT(*) as total_ok 
		FROM production_mdcw 
		WHERE is_skipped = FALSE AND reg5 IN (41, 521, 553) 
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD'), prefix
	`)
	if err == nil {
		defer rowsMdcw.Close()
		for rowsMdcw.Next() {
			var dt, pfx string
			var cnt int
			if err := rowsMdcw.Scan(&dt, &pfx, &cnt); err == nil {
				pcode := prefixToProduct[pfx]
				mdcwData[Key{Date: dt, Prefix: pfx, ProductCode: pcode}] = cnt
			}
		}
	}

	spData := make(map[Key]int)
	rowsSp, err := db.Query(`
		SELECT TO_CHAR(created_at, 'YYYY-MM-DD') as dt, session_id, data, COUNT(*) as total_scans 
		FROM production_sp 
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD'), session_id, data
	`)
	if err == nil {
		defer rowsSp.Close()
		for rowsSp.Next() {
			var dt, sess, data string
			var cnt int
			if err := rowsSp.Scan(&dt, &sess, &data, &cnt); err == nil {
				pfx := sessionToPrefix[sess]
				if pfx == "" {
					pfx = sess
				}
				spData[Key{Date: dt, Prefix: pfx, ProductCode: data}] = cnt
			}
		}
	}

	keysMap := make(map[Key]bool)
	for k := range mdcwData {
		keysMap[k] = true
	}
	for k := range spData {
		keysMap[k] = true
	}

	var comparison []ComparisonRow
	for k := range keysMap {
		mCount := mdcwData[k]
		sCount := spData[k]

		prod := prods[k.ProductCode]
		qtyPack := prod.QtyPack
		if qtyPack <= 0 {
			qtyPack = 1
		}

		spPacks := sCount * qtyPack
		disc := mCount - spPacks
		discPct := 0.0
		if mCount > 0 {
			discPct = (float64(disc) / float64(mCount)) * 100
		} else if spPacks > 0 {
			discPct = -100.0
		}

		line := ""
		if strings.Contains(k.Prefix, "MDCW1") || strings.Contains(k.Prefix, "SP1") {
			line = "Line 1"
		} else if strings.Contains(k.Prefix, "MDCW2") || strings.Contains(k.Prefix, "SP2") {
			line = "Line 2"
		} else if strings.Contains(k.Prefix, "MDCW3") || strings.Contains(k.Prefix, "SP3") {
			line = "Line 3"
		} else {
			line = k.Prefix
		}

		comparison = append(comparison, ComparisonRow{
			Date:           k.Date,
			Line:           line,
			ProductCode:    k.ProductCode,
			ProductName:    prod.Name,
			MdcwPacks:      mCount,
			SpCartons:      sCount,
			SpPacks:        spPacks,
			QtyPack:        qtyPack,
			Discrepancy:    disc,
			DiscrepancyPct: discPct,
		})
	}

	return comparison, nil
}
