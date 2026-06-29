package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	hst := getEnv("DB_HOST", "postgres_db")
	prt := getEnv("DB_PORT", "5432")
	usr := getEnv("DB_USER", "postgres")
	pwd := getEnv("DB_PASSWORD", "password_rahasia_anda")
	nm := getEnv("DB_NAME", "servfi")

	cnn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", usr, pwd, hst, prt, nm)

	var err error
	db, err = sql.Open("postgres", cnn)
	if err != nil {
		hndlErr("initDB.Open", err)
	} else if err = db.Ping(); err != nil {
		hndlErr("initDB.Ping", err)
	} else {
		lg("Connected to PostgreSQL")
	}
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}
