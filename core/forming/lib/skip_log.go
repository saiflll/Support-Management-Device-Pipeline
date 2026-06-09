package lib

import (
	"database/sql"
	"log"
)

// Skip data dengan weight = 0 dan log
func LogSkippedData(db *sql.DB, ts string, reg2, reg5, reg114 int, prf string) {
	if db == nil {
		return
	}

	log.Printf("[SKIP] Data with weight=0 from %s at %s (reg2=%d, reg5=%d)", prf, ts, reg2, reg5)

	qry := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(qry, ts, reg2, reg5, reg114, prf, "Weight is zero")
	if err != nil {
		log.Println("Error logging skipped data:", err)
	}
}

func GetSkipLogs(db *sql.DB) ([]map[string]interface{}, error) {
	if db == nil {
		return []map[string]interface{}{}, nil
	}

	qry := `SELECT id, ts, reg2, reg5, reg114, prefix, reason, skipped_at 
	          FROM skip_log 
	          ORDER BY skipped_at DESC 
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
