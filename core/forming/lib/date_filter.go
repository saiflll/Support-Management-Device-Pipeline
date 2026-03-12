package lib

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

func NormalizeRecord(prefix string, reg5 int, reg114 int) (string, int, int) {
	if prefix == "" {
		return prefix, reg5, reg114
	}

	// Convert negative weight (reg114) to positive
	if reg114 < 0 {
		reg114 = -reg114
	}

	prefixNormalized := strings.ToUpper(strings.ReplaceAll(prefix, " ", ""))

	if prefixNormalized == "MDCW" || strings.Contains(prefixNormalized, "TEST") || strings.Contains(prefixNormalized, "LINE2") || strings.Contains(prefixNormalized, "CEK") {
		return "IGNORE_RECORD", reg5, reg114
	}

	if strings.Contains(prefixNormalized, "MDCW1") || strings.Contains(prefixNormalized, "(UK)") {
		prefix = "MDCW1 (UK)"
		if reg5 != 8201 {
			if reg114 < 8710 {
				reg5 = 25
			} else if reg114 > 9520 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW2") || strings.Contains(prefixNormalized, "SIOMAY") {
		prefix = "MDCW2 (Siomay)"
		if reg5 != 8201 {
			if reg114 < 7040 {
				reg5 = 25
			} else if reg114 > 7540 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW3") || strings.Contains(prefixNormalized, "PENTOL") {
		prefix = "MDCW3 (Pentol)"
		if reg5 != 8201 {
			if reg114 < 5840 {
				reg5 = 25
			} else if reg114 > 6340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW4") || strings.Contains(prefixNormalized, "AP") {
		prefix = "MDCW4 (AP)"
		if reg5 != 8201 {
			if reg114 < 14940 {
				reg5 = 25
			} else if reg114 > 15660 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW5") || strings.Contains(prefixNormalized, "ACIN") {
		prefix = "MDCW5 (ACIN)"
		if reg5 != 8201 {
			if reg114 < 10100 {
				reg5 = 25
			} else if reg114 > 10270 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW6") || strings.Contains(prefixNormalized, "LUMPIA") {
		prefix = "MDCW6 (Lumpia)"
		if reg5 != 8201 {
			if reg114 < 3080 {
				reg5 = 25
			} else if reg114 > 3340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prefixNormalized, "MDCW") {
		// Gabungkan MDCW lainnya yang punya spasi/huruf kecil
		prefix = prefixNormalized
		if reg5 != 8201 && (reg5 == 9 || reg5 == 90 || reg5 == 8 || reg5 == 0) && reg114 > 0 {
			reg5 = 41 // Paksa menjadi OK secara visual jika masih IDLE/MATI tapi punya berat > 0
		}
	} else {
		prefix = strings.ToUpper(strings.TrimSpace(prefix))
	}
	return prefix, reg5, reg114
}

func MatchStatusFilter(statusFilter string, reg5 int) bool {
	if statusFilter == "" || statusFilter == "all" {
		return true
	}
	switch statusFilter {
	case "ok":
		return reg5 == 41 || reg5 == 521 || reg5 == 553
	case "idle":
		return reg5 == 9 || reg5 == 90
	case "metal":
		return reg5 == 8201
	case "under":
		return reg5 == 25
	case "over":
		return reg5 == 73
	case "mati":
		return reg5 == 8
	case "unknown":
		return reg5 != 8 && reg5 != 9 && reg5 != 90 && reg5 != 41 && reg5 != 521 && reg5 != 553 && reg5 != 8201 && reg5 != 25 && reg5 != 73
	default:
		return fmt.Sprintf("%d", reg5) == statusFilter
	}
}

// Record structure for the DB
type Record struct {
	ID              int       `json:"id"`
	Ts              string    `json:"ts"`
	Reg2            int       `json:"reg2"`
	Reg5            int       `json:"reg5"`
	Reg114          int       `json:"reg114"`
	WeightFormatted string    `json:"weight_formatted"`
	Prefix          string    `json:"prefix"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetRecordsByDateRange(db *sql.DB, startDate, endDate, prefixFilter, statusFilter, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
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

	// Sorting
	if sortBy == "weight_desc" {
		query += " ORDER BY reg114 DESC"
	} else if sortBy == "weight_asc" {
		query += " ORDER BY reg114 ASC"
	} else {
		query += " ORDER BY created_at DESC"
	}

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
		} else {
			r.Reg114 = 0
		}

		r.Prefix, r.Reg5, r.Reg114 = NormalizeRecord(r.Prefix, r.Reg5, r.Reg114)

		if r.Prefix == "IGNORE_RECORD" {
			continue
		}

		if prefixFilter != "" && prefixFilter != "all" && r.Prefix != prefixFilter {
			continue
		}

		if !MatchStatusFilter(statusFilter, r.Reg5) {
			continue
		}

		if seenPrefix[r.Prefix] && lastPackCnt[r.Prefix] == r.Reg2 {
			continue
		}
		seenPrefix[r.Prefix] = true
		lastPackCnt[r.Prefix] = r.Reg2

		intPart := r.Reg114 / 10
		decPart := r.Reg114 % 10
		r.WeightFormatted = fmt.Sprintf("%d,%d g", intPart, decPart)

		records = append(records, r)
	}

	return records, nil
}

// GetPrefixes returns list of unique prefixes
func GetPrefixes(db *sql.DB) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	query := `SELECT DISTINCT prefix FROM production_mdcw WHERE prefix IS NOT NULL`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error fetching prefixes: %v", err)
		return []string{}, nil
	}
	defer rows.Close()

	prefixMap := make(map[string]bool)
	for rows.Next() {
		var prefix string
		if err := rows.Scan(&prefix); err != nil {
			continue
		}
		if prefix != "" {
			norm, _, _ := NormalizeRecord(prefix, 0, 0)
			if norm != "IGNORE_RECORD" {
				prefixMap[norm] = true
			}
		}
	}

	var prefixes []string
	for p := range prefixMap {
		prefixes = append(prefixes, p)
	}

	for i := 0; i < len(prefixes); i++ {
		for j := i + 1; j < len(prefixes); j++ {
			if prefixes[i] > prefixes[j] {
				prefixes[i], prefixes[j] = prefixes[j], prefixes[i]
			}
		}
	}

	return prefixes, nil
}
