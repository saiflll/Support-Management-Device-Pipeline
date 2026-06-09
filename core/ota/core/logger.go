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

func Lg(f string, v ...interface{}) {
	if DbgMde {
		log.Printf(f, v...)
	}
}

func HndlErr(c string, err error) {
	log.Printf("❌ [%s] ERROR: %v", c, err)
}

func Ftl(f string, v ...interface{}) {
	log.Fatalf("❌ FATAL: "+f, v...)
}
