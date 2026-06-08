package logger

import (
	"log"
	"os"
)

var DbgMde = true // DebugMode (default = true)

// AtrLogger (Atur Logger) initializes the debug mode state from environment variables
func AtrLogger() {
	dm := os.Getenv("DEBUG_MODE")
	if dm == "0" || dm == "off" {
		DbgMde = false
	}
}

// Lg (Log) prints normal log messages if DbgMde is enabled
func Lg(format string, v ...interface{}) {
	if DbgMde {
		log.Printf(format, v...)
	}
}

// HndlErr (Handle Error) handles error logging centrally
func HndlErr(ctx string, err error) {
	log.Printf("❌ [%s] ERROR: %v", ctx, err)
}

// Ftl (Fatal) prints fatal message and exits
func Ftl(format string, v ...interface{}) {
	log.Fatalf("❌ FATAL: "+format, v...)
}
