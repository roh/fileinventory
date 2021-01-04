package models

import (
	"database/sql"
	"log"
	"time"
)

// FoundFile ...
type FoundFile struct {
	Source      string
	Path        string
	Md5hash     string
	Name        string
	Extension   string
	Type        string
	Size        int64
	Modified    time.Time
	Discovered  time.Time
	LastChecked time.Time
}

// CreateFoundFileTable ...
func CreateFoundFileTable(db *sql.DB) {
	const sql = `
		CREATE TABLE if not exists found_files (
			source TEXT NOT NULL,
			path TEXT NOT NULL,
			md5hash TEXT NOT NULL,
			name TEXT NOT NULL,
			size int NOT NULL,
			modified TIMESTAMP NOT NULL,
			extension TEXT,
			type TEXT,
			classification TEXT,
			tags TEXT,
			discovered TIMESTAMP NOT NULL,
			last_checked TIMESTAMP NOT NULL,
			unique(source, path, md5hash)
	    )`
	_, err := db.Exec(sql)
	if err != nil {
		log.Fatal(err)
	}
}

// Save ...
func (ff *FoundFile) Save(db *sql.DB) {
	// If the file changes, it is considered a different file, even if it is in the same path.
	// FIXME: Need to select before doing an upsert, since file may already be discovered, causing the discovered field to have side effects that aren't good
	const sql = `
		INSERT INTO found_files (source, path, md5hash, name, extension, type, size, modified, discovered, last_checked)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (source, path, md5hash) DO UPDATE SET
			name=excluded.name,
			type=excluded.type,
			extension=excluded.extension,
			size=excluded.size,
			modified=excluded.modified,
			last_checked=excluded.last_checked`
	_, err := db.Exec(sql, ff.Source, ff.Path, ff.Md5hash, ff.Name, ff.Extension, ff.Type, ff.Size, ff.Modified, ff.Discovered, ff.LastChecked)
	if err != nil {
		log.Panic(err)
	}
}
