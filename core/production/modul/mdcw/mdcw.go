package mdcw

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"production/lib"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	db            *sql.DB
	mqttClient    mqtt.Client
	lastPayloads   = make(map[string]Payload)
	lastPayloadsMu sync.Mutex
)

func Init(database *sql.DB, mqttCli mqtt.Client) {
	db = database
	mqttClient = mqttCli

	createTable()
	ensureColumns()

	if mqttClient != nil {
		subscribe(mqttClient)
	}
}

func createTable() {
	qry := `
	CREATE TABLE IF NOT EXISTS production_mdcw (
		id SERIAL PRIMARY KEY,
		ts VARCHAR(50),
		reg2 INTEGER,
		reg5 INTEGER,
		reg114 INTEGER,
		prefix VARCHAR(50),
		data_type VARCHAR(20) DEFAULT 'VALID',
		confidence REAL DEFAULT 1.0,
		is_skipped BOOLEAN DEFAULT FALSE,
		skip_reason VARCHAR(100) DEFAULT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(qry); err != nil {
		log.Printf("[MDCW] createTable error: %v", err)
	} else {
		log.Println("[MDCW] Table 'production_mdcw' ensured")
	}
}

func ensureColumns() {
	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS data_type VARCHAR(20) DEFAULT 'VALID'"); err != nil {
		log.Printf("[MDCW] ensureColumns data_type: %v", err)
	} else {
		log.Println("[MDCW] Column 'data_type' ensured")
	}

	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS confidence REAL DEFAULT 1.0"); err != nil {
		log.Printf("[MDCW] ensureColumns confidence: %v", err)
	} else {
		log.Println("[MDCW] Column 'confidence' ensured")
	}

	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS is_skipped BOOLEAN DEFAULT FALSE"); err != nil {
		log.Printf("[MDCW] ensureColumns is_skipped: %v", err)
	}

	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS skip_reason VARCHAR(100) DEFAULT NULL"); err != nil {
		log.Printf("[MDCW] ensureColumns skip_reason: %v", err)
	}
}

func subscribe(cln mqtt.Client) {
	tpc := "production/mdcw"
	if tkn := cln.Subscribe(tpc, 1, messageHandler); tkn.Wait() && tkn.Error() != nil {
		log.Printf("[MDCW] Error subscribing to topic %s: %v", tpc, tkn.Error())
	} else {
		log.Printf("[MDCW] Subscribed to topic: %s", tpc)
	}
}

func messageHandler(cln mqtt.Client, psn mqtt.Message) {
	var pl Payload
	if err := json.Unmarshal(psn.Payload(), &pl); err != nil {
		log.Printf("[MDCW] Error parsing JSON: %v", err)
		return
	}

	if pl.Prefix == "" && pl.NodePrefix != "" {
		pl.Prefix = pl.NodePrefix
	}

	if pl.Reg2 == 0 && pl.Data.Reg2 != 0 {
		pl.Reg2 = pl.Data.Reg2
	}
	if pl.Reg5 == 0 && pl.Data.Reg5 != 0 {
		pl.Reg5 = pl.Data.Reg5
	}
	if pl.Reg114 == 0 && pl.Data.Reg114 != 0 {
		pl.Reg114 = pl.Data.Reg114
	}

	if pl.Reg2 == 0 && pl.Total != 0 {
		pl.Reg2 = pl.Total
	}
	if pl.Reg5 == 0 && pl.Code != 0 {
		pl.Reg5 = pl.Code
	}
	if pl.Reg114 == 0 && pl.Weight != 0 {
		pl.Reg114 = pl.Weight
	}

	wkt := time.Now().Format("2006-01-02 15:04:05")
	if pl.Ts == nil {
		pl.Ts = wkt
	} else if _, ok := pl.Ts.(string); !ok {
		pl.Ts = wkt
	}

	insertData(pl)
}

func insertData(psn Payload) {
	if db == nil {
		return
	}

	ts := fmt.Sprintf("%v", psn.Ts)
	psn.Prefix, psn.Reg5, psn.Reg114 = NormalizeRecord(psn.Prefix, psn.Reg5, psn.Reg114)

	if psn.Prefix == "IGNORE_RECORD" {
		return
	}

	lastPayloadsMu.Lock()
	prv, ada := lastPayloads[psn.Prefix]

	isDpl := false
	if ada && prv.Reg2 == psn.Reg2 {
		isDpl = true
	} else if !ada {
		var reg2 int
		err := db.QueryRow("SELECT reg2 FROM production_mdcw WHERE prefix = $1 ORDER BY id DESC LIMIT 1", psn.Prefix).Scan(&reg2)
		if err == nil && reg2 == psn.Reg2 {
			isDpl = true
		}
	}

	if !isDpl {
		lastPayloads[psn.Prefix] = psn
	}
	lastPayloadsMu.Unlock()

	if isDpl {
		log.Printf("[MDCW] [SKIP] Duplicate data from %s: reg2=%d", psn.Prefix, psn.Reg2)
		qry := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix, is_skipped, skip_reason) VALUES ($1, $2, $3, $4, $5, TRUE, $6)`
		if _, err := db.Exec(qry, ts, psn.Reg2, psn.Reg5, psn.Reg114, psn.Prefix, "Duplicate data (reg2 unchanged)"); err != nil {
			log.Printf("[MDCW] insertData: log skipped data (duplicate): %v", err)
		}
		return
	}

	typ, cfd := AnalyzeRecord(psn.Prefix, psn.Reg114)

	if typ == DataTypeSpam && cfd > 0.8 {
		log.Printf("[MDCW] [FILTER] Spam detected from %s (delay too low)", psn.Prefix)
		qry := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix, data_type, confidence, is_skipped, skip_reason) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8)`
		if _, err := db.Exec(qry, ts, psn.Reg2, psn.Reg5, psn.Reg114, psn.Prefix, typ, cfd, "Spam detection (ML)"); err != nil {
			log.Printf("[MDCW] insertData: log skipped data (spam): %v", err)
		}
		return
	}

	qry := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix, data_type, confidence) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := db.Exec(qry, ts, psn.Reg2, psn.Reg5, psn.Reg114, psn.Prefix, typ, cfd); err != nil {
		log.Printf("[MDCW] insertData: production_mdcw: %v", err)
	} else {
		log.Printf("[MDCW] Data inserted (Type: %s, Conf: %.2f)", typ, cfd)

		go func(pl Payload, dTpe string) {
			if err := lib.AppendToSheet(lib.Payload{
				Ts:     fmt.Sprintf("%v", pl.Ts),
				Reg2:   pl.Reg2,
				Reg5:   pl.Reg5,
				Reg114: pl.Reg114,
				Prefix: pl.Prefix + " [" + dTpe + "]",
			}); err != nil {
				log.Printf("[MDCW] Failed to export to Sheets: %v", err)
			}
		}(psn, typ)

		go ForwardToCloud(psn, psn.Prefix, psn.Reg5, psn.Reg114)
	}
}

func GetRecords(pfxFltr, stsFltr, srtBy, mliWkt, hntWkt string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	qry := "SELECT id, ts, reg2, reg5, reg114, prefix, data_type, confidence, created_at FROM production_mdcw WHERE is_skipped = FALSE"
	var args []interface{}
	argId := 1

	if pfxFltr != "" && pfxFltr != "all" {
		qry += fmt.Sprintf(" AND UPPER(REPLACE(prefix, ' ', '')) = $%d", argId)
		args = append(args, strings.ToUpper(strings.ReplaceAll(pfxFltr, " ", "")))
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

	switch srtBy {
	case "weight_desc":
		qry += " ORDER BY reg114 DESC"
	case "weight_asc":
		qry += " ORDER BY reg114 ASC"
	default:
		qry += " ORDER BY created_at DESC"
	}
	qry += " LIMIT 2000"

	rows, err := db.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Record
	lstPack := make(map[string]int)
	sn := make(map[string]bool)

	for rows.Next() {
		var r Record
		var prf sql.NullString
		var r2, r5, r114 sql.NullInt64
		var dTpe sql.NullString
		var conf sql.NullFloat64

		if err := rows.Scan(&r.ID, &r.Ts, &r2, &r5, &r114, &prf, &dTpe, &conf, &r.CreatedAt); err != nil {
			return nil, err
		}

		r.DataType = DataTypeValid
		if dTpe.Valid {
			r.DataType = dTpe.String
		}
		if conf.Valid {
			r.Confidence = conf.Float64
		}

		r.Prefix = "-"
		if prf.Valid {
			r.Prefix = prf.String
		}
		if r2.Valid {
			r.Reg2 = int(r2.Int64)
		}
		if r5.Valid {
			r.Reg5 = int(r5.Int64)
		}

		if r114.Valid {
			r.Reg114 = int(r114.Int64)
		} else {
			r.Reg114 = 0
		}

		if r.Prefix == "IGNORE_RECORD" || !MatchStatusFilter(stsFltr, r.Reg5) {
			continue
		}

		if sn[r.Prefix] && lstPack[r.Prefix] == r.Reg2 {
			continue
		}
		sn[r.Prefix] = true
		lstPack[r.Prefix] = r.Reg2

		r.WeightFormatted = fmt.Sprintf("%d,%d g", r.Reg114/10, r.Reg114%10)
		recs = append(recs, r)

		if len(recs) >= 100 {
			break
		}
	}
	return recs, nil
}

func GetSummary() ([]Summary, error) {
	if db == nil {
		return []Summary{}, nil
	}

	qry := `
		SELECT prefix, reg5, reg114
		FROM production_mdcw
		WHERE DATE(created_at) = CURRENT_DATE AND is_skipped = FALSE
		ORDER BY created_at DESC
	`
	rows, err := db.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	smr := make(map[string]*Summary)

	for rows.Next() {
		var prefix sql.NullString
		var reg5, reg114 sql.NullInt64

		if err := rows.Scan(&prefix, &reg5, &reg114); err != nil {
			continue
		}

		prf, st, wt := "", 0, 0
		if prefix.Valid {
			prf = prefix.String
		}
		if reg5.Valid {
			st = int(reg5.Int64)
		}
		if reg114.Valid {
			wt = int(reg114.Int64)
		}

		npfx, nreg5, nreg114 := NormalizeRecord(prf, st, wt)
		if npfx == "IGNORE_RECORD" || npfx == "" || npfx == "-" {
			continue
		}

		if _, ada := smr[npfx]; !ada {
			smr[npfx] = &Summary{Prefix: npfx, MinWeight: nreg114}
		}

		s := smr[npfx]
		s.TotalCount++

		isOk := nreg5 == 41 || nreg5 == 521 || nreg5 == 553
		if isOk {
			s.OkCount++
			s.OkWeight += nreg114
		} else if nreg5 == 25 {
			s.UnderCount++
		} else if nreg5 == 73 {
			s.OverCount++
		} else if nreg5 == 8201 {
			s.MetalCount++
		}

		s.AvgWeight += nreg114
		s.SumWeight += nreg114

		if nreg114 > 0 {
			if s.MinWeight == 0 || nreg114 < s.MinWeight {
				s.MinWeight = nreg114
			}
			if nreg114 > s.MaxWeight {
				s.MaxWeight = nreg114
			}
		}
	}

	var smrs []Summary
	for _, s := range smr {
		if s.TotalCount > 0 {
			s.AvgWeight /= s.TotalCount
		}
		smrs = append(smrs, *s)
	}

	for i := 0; i < len(smrs); i++ {
		for j := i + 1; j < len(smrs); j++ {
			if smrs[i].Prefix > smrs[j].Prefix {
				smrs[i], smrs[j] = smrs[j], smrs[i]
			}
		}
	}

	return smrs, nil
}

func GetPrefixes() ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	qry := `SELECT DISTINCT prefix FROM production_mdcw WHERE prefix IS NOT NULL`
	rows, err := db.Query(qry)
	if err != nil {
		return []string{}, nil
	}
	defer rows.Close()

	mapPrf := make(map[string]bool)
	for rows.Next() {
		var prf string
		if err := rows.Scan(&prf); err != nil {
			continue
		}
		if prf != "" {
			norm, _, _ := NormalizeRecord(prf, 0, 0)
			if norm != "IGNORE_RECORD" {
				mapPrf[norm] = true
			}
		}
	}

	var prfs []string
	for p := range mapPrf {
		prfs = append(prfs, p)
	}

	for i := 0; i < len(prfs); i++ {
		for j := i + 1; j < len(prfs); j++ {
			if prfs[i] > prfs[j] {
				prfs[i], prfs[j] = prfs[j], prfs[i]
			}
		}
	}

	return prfs, nil
}

func GetRecordsByDateRange(tglMli, tglHnt, fltPrf, fltSts, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	qry := "SELECT id, ts, reg2, reg5, reg114, prefix, data_type, confidence, created_at FROM production_mdcw WHERE is_skipped = FALSE"
	args := []interface{}{}
	argId := 1

	if tglMli != "" {
		qry += fmt.Sprintf(" AND DATE(created_at) >= $%d", argId)
		args = append(args, tglMli)
		argId++
	}
	if tglHnt != "" {
		qry += fmt.Sprintf(" AND DATE(created_at) <= $%d", argId)
		args = append(args, tglHnt)
		argId++
	}

	if sortBy == "weight_desc" {
		qry += " ORDER BY reg114 DESC"
	} else if sortBy == "weight_asc" {
		qry += " ORDER BY reg114 ASC"
	} else {
		qry += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Record
	lstPack := make(map[string]int)
	snPrf := make(map[string]bool)
	for rows.Next() {
		var r Record
		var prf sql.NullString
		var reg2, reg5, reg114 sql.NullInt64
		var dTpe sql.NullString
		var conf sql.NullFloat64

		if err := rows.Scan(&r.ID, &r.Ts, &reg2, &reg5, &reg114, &prf, &dTpe, &conf, &r.CreatedAt); err != nil {
			return nil, err
		}

		if dTpe.Valid {
			r.DataType = dTpe.String
		} else {
			r.DataType = "VALID"
		}
		if conf.Valid {
			r.Confidence = conf.Float64
		} else {
			r.Confidence = 1.0
		}

		if prf.Valid {
			r.Prefix = prf.String
		} else {
			r.Prefix = "-"
		}

		if reg2.Valid {
			r.Reg2 = int(reg2.Int64)
		}
		if reg5.Valid {
			r.Reg5 = int(reg5.Int64)
		}
		if reg114.Valid {
			r.Reg114 = int(reg114.Int64)
		} else {
			r.Reg114 = 0
		}

		r.Prefix, r.Reg5, r.Reg114 = NormalizeRecord(r.Prefix, r.Reg5, r.Reg114)

		if r.Prefix == "IGNORE_RECORD" {
			continue
		}

		if fltPrf != "" && fltPrf != "all" && r.Prefix != fltPrf {
			continue
		}

		if !MatchStatusFilter(fltSts, r.Reg5) {
			continue
		}

		if snPrf[r.Prefix] && lstPack[r.Prefix] == r.Reg2 {
			continue
		}
		snPrf[r.Prefix] = true
		lstPack[r.Prefix] = r.Reg2

		intPrt := r.Reg114 / 10
		decPrt := r.Reg114 % 10
		r.WeightFormatted = fmt.Sprintf("%d,%d g", intPrt, decPrt)

		recs = append(recs, r)
	}

	return recs, nil
}

func GetSkipLogs() ([]map[string]interface{}, error) {
	if db == nil {
		return []map[string]interface{}{}, nil
	}

	qry := `SELECT id, ts, reg2, reg5, reg114, prefix, skip_reason as reason, created_at as skipped_at 
	          FROM production_mdcw 
	          WHERE is_skipped = TRUE
	          ORDER BY created_at DESC 
	          LIMIT 100`

	rows, err := db.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lgs []map[string]interface{}
	for rows.Next() {
		var id int
		var ts, prf, rsn string
		var reg2, reg5, reg114 int
		var skpWkt sql.NullTime

		if err := rows.Scan(&id, &ts, &reg2, &reg5, &reg114, &prf, &rsn, &skpWkt); err != nil {
			return nil, err
		}

		ent := map[string]interface{}{
			"id":         id,
			"ts":         ts,
			"reg2":       reg2,
			"reg5":       reg5,
			"reg114":     reg114,
			"prefix":     prf,
			"reason":     rsn,
			"skipped_at": skpWkt.Time,
		}
		lgs = append(lgs, ent)
	}

	return lgs, nil
}

type DailyStat struct {
	Date       string `json:"date"`
	TotalCount int    `json:"total_count"`
	OkCount    int    `json:"ok_count"`
	UnderCount int    `json:"under_count"`
	OverCount  int    `json:"over_count"`
	MetalCount int    `json:"metal_count"`
}

func GetDailyStats(days int) ([]DailyStat, error) {
	if db == nil {
		return []DailyStat{}, nil
	}

	prm := fmt.Sprintf("%d", days)
	qry := `
		SELECT DATE(created_at) as dt, 
			COUNT(*) as total,
			SUM(CASE WHEN reg5 IN (41,521,553) THEN 1 ELSE 0 END) as ok,
			SUM(CASE WHEN reg5 = 25 THEN 1 ELSE 0 END) as under,
			SUM(CASE WHEN reg5 = 73 THEN 1 ELSE 0 END) as over,
			SUM(CASE WHEN reg5 = 8201 THEN 1 ELSE 0 END) as metal
		FROM production_mdcw
		WHERE created_at >= CURRENT_DATE - $1::INTEGER * INTERVAL '1 day' AND is_skipped = FALSE
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
		var total, ok, under, over, metal sql.NullInt64

		if err := rows.Scan(&dt, &total, &ok, &under, &over, &metal); err != nil {
			continue
		}

		if dt.Valid {
			s.Date = dt.Time.Format("2006-01-02")
		}
		if total.Valid {
			s.TotalCount = int(total.Int64)
		}
		if ok.Valid {
			s.OkCount = int(ok.Int64)
		}
		if under.Valid {
			s.UnderCount = int(under.Int64)
		}
		if over.Valid {
			s.OverCount = int(over.Int64)
		}
		if metal.Valid {
			s.MetalCount = int(metal.Int64)
		}

		stats = append(stats, s)
	}

	return stats, nil
}

func GetConveyorRecords(kodeBatch, kodeProduk string, shift int, startDate, endDate string) ([]ConveyorRecord, error) {
	if db == nil {
		return []ConveyorRecord{}, nil
	}

	qry := `SELECT c.id_record, c.factory, c.kode_produk, c.qty_per_pack, 
	               COALESCE(m.qty_pack, 0) as pack_per_karton, 
	               COALESCE(CAST(m.gram AS NUMERIC(10,2)) / NULLIF(m.qty_pack, 0), 0) as gramasi_pack, 
	               COALESCE(m.gram, 0) as gramasi_karton, 
	               c.tanggal_produksi, c.shift, 
	               c.tanggal_best_before, c.kode_ketentuan, c.kode_batch, c.tanggal_record 
	          FROM conveyor_logs c
	          LEFT JOIN master_produk m ON c.kode_produk = m.kode
	         WHERE 1=1`
	var args []interface{}
	argId := 1

	if kodeBatch != "" {
		qry += fmt.Sprintf(" AND kode_batch = $%d", argId)
		args = append(args, kodeBatch)
		argId++
	}
	if kodeProduk != "" {
		qry += fmt.Sprintf(" AND kode_produk = $%d", argId)
		args = append(args, kodeProduk)
		argId++
	}
	if shift > 0 {
		qry += fmt.Sprintf(" AND shift = $%d", argId)
		args = append(args, shift)
		argId++
	}
	if startDate != "" {
		qry += fmt.Sprintf(" AND DATE(tanggal_record) >= $%d", argId)
		args = append(args, startDate)
		argId++
	}
	if endDate != "" {
		qry += fmt.Sprintf(" AND DATE(tanggal_record) <= $%d", argId)
		args = append(args, endDate)
		argId++
	}

	qry += " ORDER BY id_record DESC LIMIT 100"

	rows, err := db.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []ConveyorRecord
	for rows.Next() {
		var r ConveyorRecord
		var fact, kProd, tglProd, tglBB, kKet, kBatch sql.NullString
		var qtyPack, packKart, gramKart, shft sql.NullInt64
		var gramPack sql.NullFloat64
		var tglRec sql.NullTime

		err := rows.Scan(
			&r.IDRecord, &fact, &kProd, &qtyPack, &packKart,
			&gramPack, &gramKart, &tglProd, &shft,
			&tglBB, &kKet, &kBatch, &tglRec,
		)
		if err != nil {
			return nil, err
		}

		if fact.Valid { r.Factory = fact.String }
		if kProd.Valid { r.KodeProduk = kProd.String }
		if qtyPack.Valid { r.QtyPerPack = int(qtyPack.Int64) }
		if packKart.Valid { r.PackPerKarton = int(packKart.Int64) }
		if gramPack.Valid { r.GramasiPack = gramPack.Float64 }
		if gramKart.Valid { r.GramasiKarton = int(gramKart.Int64) }
		if tglProd.Valid { r.TanggalProduksi = tglProd.String }
		if shft.Valid { r.Shift = int(shft.Int64) }
		if tglBB.Valid { r.TanggalBestBefore = tglBB.String }
		if kKet.Valid { r.KodeKetentuan = kKet.String }
		if kBatch.Valid { r.KodeBatch = kBatch.String }
		if tglRec.Valid { r.TanggalRecord = tglRec.Time }

		recs = append(recs, r)
	}

	return recs, nil
}

