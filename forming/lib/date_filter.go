package lib

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// GetRecordsByDateRange returns records filtered by date range
func GetRecordsByDateRange(db *sql.DB, startDate, endDate, prefixFilter, statusFilter, sortBy string) ([]interface{}, error) {
	if db == nil {
		return []interface{}{}, nil
	}

	query := "SELECT id, ts, reg2, reg5, reg114, prefix, created_at FROM production_mdcw WHERE 1=1"
	args := []interface{}{}
	argId := 1

	// Date range filter
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

	// Prefix filter (Normalized)
	if prefixFilter != "" && prefixFilter != "all" {
		query += fmt.Sprintf(" AND UPPER(REPLACE(prefix, ' ', '')) = $%d", argId)
		args = append(args, prefixFilter)
		argId++
	}

	// Status filter
	if statusFilter != "" && statusFilter != "all" {
		query += fmt.Sprintf(" AND reg5 = $%d", argId)
		args = append(args, statusFilter)
		argId++
	}

	// Sorting
	if sortBy == "weight_desc" {
		query += " ORDER BY reg114 DESC"
	} else if sortBy == "weight_asc" {
		query += " ORDER BY reg114 ASC"
	} else {
		query += " ORDER BY created_at DESC"
	}

	query += " LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type Record struct {
		ID              int       `json:"id"`
		Ts              string    `json:"ts"`
		Reg2            int       `json:"reg2"`
		Reg5            int       `json:"reg5"`
		Reg114          int       `json:"reg114"`
		WeightFormatted string    `json:"weight_formatted"` // Added formatted weight
		Prefix          string    `json:"prefix"`
		CreatedAt       time.Time `json:"created_at"`
	}

	var records []interface{}
	for rows.Next() {
		var r Record
		var prefix sql.NullString
		var reg2, reg5, reg114 sql.NullInt64

		if err := rows.Scan(&r.ID, &r.Ts, &reg2, &reg5, &reg114, &prefix, &r.CreatedAt); err != nil {
			return nil, err
		}

		if prefix.Valid {
			r.Prefix = prefix.String
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
			// Format weight: divide by 10 for 1 decimal place (shift decimal left)
			intPart := r.Reg114 / 10
			decPart := r.Reg114 % 10
			r.WeightFormatted = fmt.Sprintf("%d,%d g", intPart, decPart)
		} else {
			r.WeightFormatted = "0,0 g"
		}

		records = append(records, r)
	}

	return records, nil
}

// GetPrefixes returns list of unique prefixes
func GetPrefixes(db *sql.DB) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	// Get list of unique prefixes, normalized (UPPERCASE, NO SPACES)
	query := `SELECT DISTINCT UPPER(REPLACE(prefix, ' ', '')) FROM production_mdcw WHERE prefix IS NOT NULL ORDER BY 1 ASC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error fetching prefixes: %v", err)
		return []string{}, nil
	}
	defer rows.Close()

	var prefixes []string
	for rows.Next() {
		var prefix string
		if err := rows.Scan(&prefix); err != nil {
			continue
		}
		if prefix != "" {
			prefixes = append(prefixes, prefix)
		}
	}

	return prefixes, nil
}
