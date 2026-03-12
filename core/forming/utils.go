package main

import "os"

// getEnv fetches an env variable, or falls back to a default value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
