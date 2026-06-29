package main

import (
	"os"
	"time"
)

// getEnv fetches an env variable, or falls back to a default value
func getEnv(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return fb
}

// timeNow mengembalikan tanggal hari ini dalam format YYYY-MM-DD (WIB)
func timeNow() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("2006-01-02")
}
