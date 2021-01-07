package models

import (
	"database/sql"
	"fmt"
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
	fmt.Println(dbPath)
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
