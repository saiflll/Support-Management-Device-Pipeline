package main

import (
	"database/sql"
	"fmt"
	"forming/lib"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	host := getEnv("DB_HOST", "postgres_db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "password_rahasia_anda")
	name := getEnv("DB_NAME", "servfi")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database connection: %v", err)
	} else if err = db.Ping(); err != nil {
		log.Printf("Warning: Could not connect to database: %v", err)
	} else {
		log.Println("Connected to PostgreSQL")
		createTable()
	}
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS production_mdcw (
		id SERIAL PRIMARY KEY,
		ts VARCHAR(50),
		reg2 INTEGER,
		reg5 INTEGER,
		reg114 INTEGER,
		prefix VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(query); err != nil {
		log.Println("Error creating table:", err)
	} else {
		log.Println("Table 'production_mdcw' ensured")
	}

	skipLogQuery := `
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
	if _, err := db.Exec(skipLogQuery); err != nil {
		log.Println("Error creating skip_log table:", err)
	} else {
		log.Println("Table 'skip_log' ensured")
	}
}

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
	prev, exists := lastPayloads[p.Prefix]

	isDuplicate := false
	if exists && prev.Reg2 == p.Reg2 {
		isDuplicate = true
	} else if !exists {
		var lastReg2 int
		errCheck := db.QueryRow("SELECT reg2 FROM production_mdcw WHERE prefix = $1 ORDER BY id DESC LIMIT 1", p.Prefix).Scan(&lastReg2)
		if errCheck == nil && lastReg2 == p.Reg2 {
			isDuplicate = true
		}
	}

	if !isDuplicate {
		lastPayloads[p.Prefix] = p
	}
	lastPayloadsMu.Unlock()

	if isDuplicate {
		log.Printf("[SKIP] Duplicate data from %s: reg2=%d", p.Prefix, p.Reg2)
		skipQuery := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := db.Exec(skipQuery, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix, "Duplicate data (reg2 unchanged)"); err != nil {
			log.Println("Error logging skipped data:", err)
		}
		return
	}

	if p.Reg114 == 0 {
		log.Printf("[SKIP] Data with weight=0 from %s at %s (reg2=%d, reg5=%d)", p.Prefix, tsStr, p.Reg2, p.Reg5)
		skipQuery := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := db.Exec(skipQuery, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix, "Weight is zero"); err != nil {
			log.Println("Error logging skipped data:", err)
		}
		return
	}

	query := `INSERT INTO production_mdcw (ts, reg2, reg5, reg114, prefix) VALUES ($1, $2, $3, $4, $5)`
	if _, err := db.Exec(query, tsStr, p.Reg2, p.Reg5, p.Reg114, p.Prefix); err != nil {
		log.Println("Error inserting data:", err)
	} else {
		log.Println("Data inserted successfully")
		go func(p Payload) {
			if err := lib.AppendToSheet(lib.Payload{
				Ts:     fmt.Sprintf("%v", p.Ts),
				Reg2:   p.Reg2,
				Reg5:   p.Reg5,
				Reg114: p.Reg114,
				Prefix: p.Prefix,
			}); err != nil {
				log.Printf("Warning: Failed to export to Sheets: %v", err)
			}
		}(p)
	}
}

func getRecords(prefixFilter, statusFilter, sortBy, startDate, endDate string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	query := "SELECT id, ts, reg2, reg5, reg114, prefix, created_at FROM production_mdcw WHERE 1=1"
	var args []interface{}
	argId := 1

	if prefixFilter != "" && prefixFilter != "all" {
		query += fmt.Sprintf(" AND UPPER(REPLACE(prefix, ' ', '')) = $%d", argId)
		args = append(args, strings.ToUpper(strings.ReplaceAll(prefixFilter, " ", "")))
		argId++
	}

	if startDate != "" {
		query += fmt.Sprintf(" AND DATE(created_at) >= $%d", argId)
		args = append(args, startDate)
		argId++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND DATE(created_at) <= $%d", argId)
		args = append(args, endDate)
		argId++
	}

	switch sortBy {
	case "weight_desc":
		query += " ORDER BY reg114 DESC"
	case "weight_asc":
		query += " ORDER BY reg114 ASC"
	default:
		query += " ORDER BY created_at DESC"
	}
	query += " LIMIT 2000"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	lastPackCnt := make(map[string]int)
	seenPrefix := make(map[string]bool)

	for rows.Next() {
		var r Record
		var p sql.NullString
		var r2, r5, r114 sql.NullInt64

		if err := rows.Scan(&r.ID, &r.Ts, &r2, &r5, &r114, &p, &r.CreatedAt); err != nil {
			return nil, err
		}

		r.Prefix = "-"
		if p.Valid { r.Prefix = p.String }
		if r2.Valid { r.Reg2 = int(r2.Int64) }
		if r5.Valid { r.Reg5 = int(r5.Int64) }
		
		if r114.Valid {
			r.Reg114 = int(r114.Int64)
		} else {
			r.Reg114 = 0
		}

		if r.Prefix == "IGNORE_RECORD" || !lib.MatchStatusFilter(statusFilter, r.Reg5) {
			continue
		}

		if seenPrefix[r.Prefix] && lastPackCnt[r.Prefix] == r.Reg2 {
			continue
		}
		seenPrefix[r.Prefix] = true
		lastPackCnt[r.Prefix] = r.Reg2

		r.WeightFormatted = fmt.Sprintf("%d,%d g", r.Reg114/10, r.Reg114%10)
		records = append(records, r)

		if len(records) >= 100 {
			break
		}
	}
	return records, nil
}

func getSummary() ([]Summary, error) {
	if db == nil {
		return []Summary{}, nil
	}

	query := `
		SELECT prefix, reg5, reg114
		FROM production_mdcw
		WHERE DATE(created_at) = CURRENT_DATE
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaryMap := make(map[string]*Summary)

	for rows.Next() {
		var prefix sql.NullString
		var reg5, reg114 sql.NullInt64

		if err := rows.Scan(&prefix, &reg5, &reg114); err != nil {
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

		if _, exists := summaryMap[npfx]; !exists {
			summaryMap[npfx] = &Summary{Prefix: npfx, MinWeight: nreg114}
		}

		s := summaryMap[npfx]
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

	var summaries []Summary
	for _, s := range summaryMap {
		if s.TotalCount > 0 {
			s.AvgWeight /= s.TotalCount
		}
		summaries = append(summaries, *s)
	}

	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[i].Prefix > summaries[j].Prefix {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	return summaries, nil
}
