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
