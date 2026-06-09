package lib

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

func NormalizeRecord(prf string, reg5 int, reg114 int) (string, int, int) {
	if prf == "" {
		return prf, reg5, reg114
	}

	if reg114 < 0 {
		reg114 = -reg114
	}

	prfNorm := strings.ToUpper(strings.ReplaceAll(prf, " ", ""))

	if prfNorm == "MDCW" || strings.Contains(prfNorm, "TEST") || strings.Contains(prfNorm, "LINE2") || strings.Contains(prfNorm, "CEK") {
		return "IGNORE_RECORD", reg5, reg114
	}

	if strings.Contains(prfNorm, "MDCW1") || strings.Contains(prfNorm, "(UK)") {
		prf = "MDCW1 (UK)"
		if reg5 != 8201 {
			if reg114 < 8710 {
				reg5 = 25
			} else if reg114 > 9520 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW2") || strings.Contains(prfNorm, "SIOMAY") {
		prf = "MDCW2 (Siomay)"
		if reg5 != 8201 {
			if reg114 < 7040 {
				reg5 = 25
			} else if reg114 > 7540 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW3") || strings.Contains(prfNorm, "PENTOL") {
		prf = "MDCW3 (Pentol)"
		if reg5 != 8201 {
			if reg114 < 5840 {
				reg5 = 25
			} else if reg114 > 6340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW4") || strings.Contains(prfNorm, "AP") {
		prf = "MDCW4 (AP)"
		if reg5 != 8201 {
			if reg114 < 14940 {
				reg5 = 25
			} else if reg114 > 15660 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW5") || strings.Contains(prfNorm, "ACIN") {
		prf = "MDCW5 (ACIN)"
		if reg5 != 8201 {
			if reg114 < 10100 {
				reg5 = 25
			} else if reg114 > 10270 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW6") || strings.Contains(prfNorm, "LUMPIA") {
		prf = "MDCW6 (Lumpia)"
		if reg5 != 8201 {
			if reg114 < 3080 {
				reg5 = 25
			} else if reg114 > 3340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW") {
		prf = prfNorm
		if reg5 != 8201 && (reg5 == 9 || reg5 == 90 || reg5 == 8 || reg5 == 0) && reg114 > 0 {
			reg5 = 41
		}
	} else {
		prf = strings.ToUpper(strings.TrimSpace(prf))
	}
	return prf, reg5, reg114
}

func MatchStatusFilter(fltSts string, reg5 int) bool {
	if fltSts == "" || fltSts == "all" {
		return true
	}
	switch fltSts {
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
		return fmt.Sprintf("%d", reg5) == fltSts
	}
}

type Record struct {
	ID              int       `json:"id"`
	Ts              string    `json:"ts"`
	Reg2            int       `json:"reg2"`
	Reg5            int       `json:"reg5"`
	Reg114          int       `json:"reg114"`
	WeightFormatted string    `json:"weight_formatted"`
	Prefix          string    `json:"prefix"`
	DataType        string    `json:"data_type"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetRecordsByDateRange(db *sql.DB, tglMli, tglHnt, fltPrf, fltSts, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	qry := "SELECT id, ts, reg2, reg5, reg114, prefix, data_type, confidence, created_at FROM production_mdcw WHERE 1=1"
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

func GetPrefixes(db *sql.DB) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	qry := `SELECT DISTINCT prefix FROM production_mdcw WHERE prefix IS NOT NULL`

	rows, err := db.Query(qry)
	if err != nil {
		log.Printf("Error fetching prefixes: %v", err)
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
