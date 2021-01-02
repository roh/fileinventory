package main

import (
	"crypto/md5"
	"database/sql"
	"flag"
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
	var sourceFlag = flag.String("source", "", "blah")
	flag.Parse()
	if *sourceFlag == "" {
		log.Fatal("Please specify a source flag, i.e. -source mylaptop")
	}
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
			fmt.Println("Scanning folder", name, "...")
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
			addFile(db, *sourceFlag, path, md5hash, name, info.Size(), info.ModTime(), time.Now())
			fmt.Println(md5hash, info.ModTime(), info.Size(), info.Name())
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

const tableCreateFilesSQL = `CREATE TABLE if not exists files (
    source TEXT NOT NULL,
    path TEXT NOT NULL,
    md5hash TEXT NOT NULL,
    filename TEXT NOT NULL,
    size int NOT NULL,
    modified TIMESTAMP NOT NULL,
    extension TEXT,
    filetype TEXT,
    classification TEXT,
	tags TEXT,
	discovered TIMESTAMP NOT NULL,
	last_checked TIMESTAMP NOT NULL,
	unique(source, path, md5hash)
   )`

// If the file changes, it is considered a different file, even if it is in the same path.
const insertFileSQL = `INSERT INTO files (source, path, md5hash, filename, size, modified, discovered, last_checked)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (source, path, md5hash) DO UPDATE SET
  filename=excluded.filename,
  size=excluded.size,
  modified=excluded.modified,
  last_checked=excluded.last_checked
`

func initDatabase(db *sql.DB) {
	_, err := db.Exec(tableCreateFilesSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func addFile(db *sql.DB, source string, path string, md5hash string, filename string, size int64, modified time.Time, lastChecked time.Time) {
	_, err := db.Exec(insertFileSQL, source, path, md5hash, filename, size, modified, lastChecked, lastChecked)
	if err != nil {
		log.Panic(err)
	}
}
