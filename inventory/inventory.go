package inventory

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

var db *sql.DB

// Init ...
func Init(path string) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		path = filepath.Join(homeDir, "index.db")
	}
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	CreateFoundFileTable()
}

// Close ...
func Close() {
	if db != nil {
		db.Close()
	}
}
