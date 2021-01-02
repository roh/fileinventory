package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		name := info.Name()
		if info.IsDir() {
			if strings.HasPrefix(name, ".") {
				fmt.Println("Skipping folder", name)
				return filepath.SkipDir
			}
			fmt.Println("Scanning contents in folder", name)
		} else if strings.HasPrefix(name, ".") {
			return nil
		} else {
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
			addFile(db, name, info.Size(), md5hash, info.ModTime(), path)
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
