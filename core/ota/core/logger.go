package core

import (
	"log"
	"os"
)

var DbgMde = true

func AtrLogger() {
	dm := os.Getenv("DEBUG_MODE")
	if dm == "0" || dm == "off" {
		DbgMde = false
	}
}

func Lg(format string, v ...interface{}) {
	if DbgMde {
		log.Printf(format, v...)
	}
}

func HndlErr(ctx string, err error) {
	log.Printf("❌ [%s] ERROR: %v", ctx, err)
}

func Ftl(format string, v ...interface{}) {
	log.Fatalf("❌ FATAL: "+format, v...)
}
