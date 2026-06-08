package main

import (
	"database/sql"
	"fmt"
	"forming/lib"
	"strings"

	_ "github.com/lib/pq"
)

// === KONEKSI DATABASE ===

var db *sql.DB

func initDB() {
	hst := getEnv("DB_HOST", "postgres_db")
	prt := getEnv("DB_PORT", "5432")
	usr := getEnv("DB_USER", "postgres")
	pwd := getEnv("DB_PASSWORD", "password_rahasia_anda")
	nm := getEnv("DB_NAME", "servfi")

	cnn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", usr, pwd, hst, prt, nm)

	var err error
	db, err = sql.Open("postgres", cnn)
	if err != nil {
		hndlErr("initDB.Open", err)
	} else if err = db.Ping(); err != nil {
		hndlErr("initDB.Ping", err)
	} else {
		lg("Connected to PostgreSQL")
		createTable()
		ensureColumns()
	}
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

// === SKEMA DATABASE ===

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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(qry); err != nil {
		hndlErr("createTable: production_mdcw", err)
	} else {
		lg("Table 'production_mdcw' ensured")
	}

	skpQry := `
	CREATE TABLE IF NOT EXISTS skip_log (
		id SERIAL PRIMARY KEY,
		ts VARCHAR(50),
		reg2 INTEGER,
		reg5 INTEGER,
		reg114 INTEGER,
		prefix VARCHAR(50),
		reason VARCHAR(100),
		skipped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(skpQry); err != nil {
		hndlErr("createTable: skip_log", err)
	} else {
		lg("Table 'skip_log' ensured")
	}
}

func ensureColumns() {
	// tambah kolom data_type jika belum ada
	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS data_type VARCHAR(20) DEFAULT 'VALID'"); err != nil {
		hndlErr("ensureColumns: data_type (may already exist)", err)
	} else {
		lg("Column 'data_type' ensured")
	}

	// tambah kolom confidence jika belum ada
	if _, err := db.Exec("ALTER TABLE production_mdcw ADD COLUMN IF NOT EXISTS confidence REAL DEFAULT 1.0"); err != nil {
		hndlErr("ensureColumns: confidence (may already exist)", err)
	} else {
		lg("Column 'confidence' ensured")
	}
}

// === OPERASI INSERT & RETRIEVE ===

func insertData(p Payload) {
	if db == nil {
		return
	}

	tsStr := fmt.Sprintf("%v", p.Ts)
	p.Prefix, p.Reg5, p.Reg114 = lib.NormalizeRecord(p.Prefix, p.Reg5, p.Reg114)

	if p.Prefix == "IGNORE_RECORD" {
		return
	}

	lastPayloadsMu.Lock()
	prv, ext := lastPayloads[p.Prefix]

	isDpl := false
	if ext && prv.Reg2 == p.Reg2 {
		isDpl = true
	} else if !ext {
		var lastReg2 int
		errCheck := db.QueryRow("SELECT reg2 FROM production_mdcw WHERE prefix = $1 ORDER BY id DESC LIMIT 1", p.Prefix).Scan(&lastReg2)
		if errCheck == nil && lastReg2 == p.Reg2 {
			isDpl = true
		}
	}

	if !isDpl {
		lastPayloads[p.Prefix] = p
	}
	lastPayloadsMu.Unlock()

	if isDpl {
		lg("[SKIP] Duplicate data from %s: reg2=%d", p.Prefix, p.Reg2)
		skpQry := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := db.Exec(skpQry, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix, "Duplicate data (reg2 unchanged)"); err != nil {
			hndlErr("insertData: log skipped data (duplicate)", err)
		}
		return
	}

	// ml filtering
	typ, cfd := AnalyzeRecord(p.Prefix, p.Reg114)
	
	// jika terdeteksi spam
	if typ == DataTypeSpam && cfd > 0.8 {
		lg("[FILTER] Spam detected from %s (delay too low)", p.Prefix)
		skpQry := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := db.Exec(skpQry, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix, "Spam detection (ML)"); err != nil {
			hndlErr("insertData: log skipped data (spam)", err)
		}
		return
	}

	qry := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix, data_type, confidence) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := db.Exec(qry, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix, typ, cfd); err != nil {
		hndlErr("insertData: production_mdcw", err)
	} else {
		lg("Data inserted successfully (Type: %s, Conf: %.2f)", typ, cfd)

		// forward ke Google Sheets (async)
		go func(pl Payload, dType string) {
			if err := lib.AppendToSheet(lib.Payload{
				Ts:     fmt.Sprintf("%v", pl.Ts),
				Reg2:   pl.Reg2,
				Reg5:   pl.Reg5,
				Reg114: pl.Reg114,
				Prefix: pl.Prefix + " [" + dType + "]",
			}); err != nil {
				hndlErr("Failed to export to Sheets", err)
			}
		}(p, typ)

		// forward ke Cloud MQTT Broker (async) — backend monitoring cloud akan terima di topik prod/mdcw
		go ForwardToCloud(p, p.Prefix, p.Reg5, p.Reg114)
	}
}

func getRecords(pfxFltr, stsFltr, srtBy, mliWkt, hntWkt string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	qry := "SELECT id, ts, reg2, reg5, reg114, prefix, data_type, confidence, created_at FROM production_mdcw WHERE 1=1"
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

	rws, err := db.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rws.Close()

	var rec []Record
	lastPackCnt := make(map[string]int)
	seen := make(map[string]bool)

	for rws.Next() {
		var r Record
		var p sql.NullString
		var r2, r5, r114 sql.NullInt64
		var dType sql.NullString
		var conf sql.NullFloat64

		if err := rws.Scan(&r.ID, &r.Ts, &r2, &r5, &r114, &p, &dType, &conf, &r.CreatedAt); err != nil {
			return nil, err
		}
		
		r.DataType = DataTypeValid
		if dType.Valid { r.DataType = dType.String }
		if conf.Valid { r.Confidence = conf.Float64 }

		r.Prefix = "-"
		if p.Valid { r.Prefix = p.String }
		if r2.Valid { r.Reg2 = int(r2.Int64) }
		if r5.Valid { r.Reg5 = int(r5.Int64) }
		
		if r114.Valid {
			r.Reg114 = int(r114.Int64)
		} else {
			r.Reg114 = 0
		}

		if r.Prefix == "IGNORE_RECORD" || !lib.MatchStatusFilter(stsFltr, r.Reg5) {
			continue
		}

		if seen[r.Prefix] && lastPackCnt[r.Prefix] == r.Reg2 {
			continue
		}
		seen[r.Prefix] = true
		lastPackCnt[r.Prefix] = r.Reg2

		r.WeightFormatted = fmt.Sprintf("%d,%d g", r.Reg114/10, r.Reg114%10)
		rec = append(rec, r)

		if len(rec) >= 100 {
			break
		}
	}
	return rec, nil
}

func getSummary() ([]Summary, error) {
	if db == nil {
		return []Summary{}, nil
	}

	qry := `
		SELECT prefix, reg5, reg114
		FROM production_mdcw
		WHERE DATE(created_at) = CURRENT_DATE
		ORDER BY created_at DESC
	`
	rws, err := db.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rws.Close()

	smr := make(map[string]*Summary)

	for rws.Next() {
		var prefix sql.NullString
		var reg5, reg114 sql.NullInt64

		if err := rws.Scan(&prefix, &reg5, &reg114); err != nil {
			continue
		}

		pfx, st, wt := "", 0, 0
		if prefix.Valid { pfx = prefix.String }
		if reg5.Valid { st = int(reg5.Int64) }
		if reg114.Valid { wt = int(reg114.Int64) }

		npfx, nreg5, nreg114 := lib.NormalizeRecord(pfx, st, wt)
		if npfx == "IGNORE_RECORD" || npfx == "" || npfx == "-" {
			continue
		}

		if _, exists := smr[npfx]; !exists {
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
			if s.MinWeight == 0 || nreg114 < s.MinWeight { s.MinWeight = nreg114 }
			if nreg114 > s.MaxWeight { s.MaxWeight = nreg114 }
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
