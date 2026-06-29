package main

import (
	"log"
	"os"
)

var dbgMde = true // debugMode (default = true)

// atrLogger (atur logger) initializes the debug mode state from environment variables
func atrLogger() {
	dm := os.Getenv("DEBUG_MODE")
	if dm == "0" || dm == "off" {
		dbgMde = false
	}
}

// lg (log) prints normal log messages if dbgMde is enabled
func lg(format string, v ...interface{}) {
	if dbgMde {
		log.Printf(format, v...)
	}
}

// hndlErr (handle error) handles error logging centrally
func hndlErr(ctx string, err error) {
	log.Printf("❌ [%s] ERROR: %v", ctx, err)
}

// ftl (fatal) prints fatal message and exits
func ftl(format string, v ...interface{}) {
	log.Fatalf("❌ FATAL: "+format, v...)
}
