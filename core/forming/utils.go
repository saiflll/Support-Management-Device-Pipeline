package main

import "os"

// getEnv fetches an env variable, or falls back to a default value
func getEnv(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return fb
}
