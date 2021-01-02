package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDatabase(db)

	var files []string

	root := "/Users/roh/Development/fileindexer"
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			h := md5.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
			}
			md5hash := fmt.Sprintf("%x", h.Sum(nil))
			addFile(db, info.Name(), info.Size(), md5hash, info.ModTime(), path)
			fmt.Println(md5hash, info.ModTime(), info.Size(), info.Name())
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

const tableCreateFilesSQL = `CREATE TABLE if not exists files (
    filename TEXT,
    size int,
    md5hash TEXT,
    modified timestamp,
    extension TEXT,
    path TEXT,
    source TEXT,
    filetype TEXT,
    classification TEXT,
    tags TEXT
   )`

const insertFileSQL = "INSERT INTO files (filename, size, md5hash, modified, path) VALUES (?, ?, ?, ?, ?)"

func initDatabase(db *sql.DB) {
	_, err := db.Exec(tableCreateFilesSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func addFile(db *sql.DB, filename string, size int64, md5hash string, modified time.Time, path string) {
	_, err := db.Exec(insertFileSQL, filename, size, md5hash, modified, path)
	if err != nil {
		log.Panic(err)
	}
}
