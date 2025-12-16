package lib

import (
	"database/sql"
	"log"
)

// Skip data dengan weight = 0 dan log
func LogSkippedData(db *sql.DB, ts string, reg2, reg5, reg114 int, prefix string) {
	if db == nil {
		return
	}

	log.Printf("[SKIP] Data with weight=0 from %s at %s (reg2=%d, reg5=%d)", prefix, ts, reg2, reg5)

	skipQuery := `INSERT INTO skip_log (ts, reg2, reg5, reg114, prefix, reason) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(skipQuery, ts, reg2, reg5, reg114, prefix, "Weight is zero")
	if err != nil {
		log.Println("Error logging skipped data:", err)
	}
}

// GetSkipLogs retrieves all skip logs
func GetSkipLogs(db *sql.DB) ([]map[string]interface{}, error) {
	if db == nil {
		return []map[string]interface{}{}, nil
	}

	query := `SELECT id, ts, reg2, reg5, reg114, prefix, reason, skipped_at 
	          FROM skip_log 
	          ORDER BY skipped_at DESC 
	          LIMIT 100`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var ts, prefix, reason string
		var reg2, reg5, reg114 int
		var skippedAt sql.NullTime

		if err := rows.Scan(&id, &ts, &reg2, &reg5, &reg114, &prefix, &reason, &skippedAt); err != nil {
			return nil, err
		}

		logEntry := map[string]interface{}{
			"id":         id,
			"ts":         ts,
			"reg2":       reg2,
			"reg5":       reg5,
			"reg114":     reg114,
			"prefix":     prefix,
			"reason":     reason,
			"skipped_at": skippedAt.Time,
		}
		logs = append(logs, logEntry)
	}

	return logs, nil
}
