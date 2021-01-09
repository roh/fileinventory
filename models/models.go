package models

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

var db *sql.DB

// InitDB ...
func InitDB(dbPath string) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		dbPath = filepath.Join(homeDir, "index.db")
	}
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	CreateFoundFileTable()
}

// CloseDB ...
func CloseDB() {
	if db != nil {
		db.Close()
	}
}
