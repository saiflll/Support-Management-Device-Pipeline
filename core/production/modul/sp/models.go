package sp

import "time"

type Record struct {
	ID        int       `json:"id"`
	SessionID string    `json:"session_id"`
	Data      string    `json:"data"`
	Ts        string    `json:"ts"`
	CreatedAt time.Time `json:"created_at"`
}

type Summary struct {
	SessionID  string `json:"session_id"`
	TotalCount int    `json:"total_count"`
	LastScan   string `json:"last_scan"`
	FirstScan  string `json:"first_scan"`
}

type ComparisonRow struct {
	Date           string  `json:"date"`
	Line           string  `json:"line"`
	ProductCode    string  `json:"product_code"`
	ProductName    string  `json:"product_name"`
	MdcwPacks      int     `json:"mdcw_packs"`
	SpCartons      int     `json:"sp_cartons"`
	SpPacks        int     `json:"sp_packs"`
	QtyPack        int     `json:"qty_pack"`
	Discrepancy    int     `json:"discrepancy"`
	DiscrepancyPct float64 `json:"discrepancy_pct"`
}
