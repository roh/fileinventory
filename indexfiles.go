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

	"github.com/roh/filetools/models"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var sourceFlag = flag.String("source", "", "")
	flag.Parse()
	if *sourceFlag == "" {
		log.Fatal("Please specify a source flag, i.e. -source mylaptop")
	}
	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	models.CreateFoundFileTable(db)

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
			foundFile := models.FoundFile{Source: *sourceFlag, Path: path, Md5hash: md5hash, Name: name, Extension: GetNormalizedExtension(path), Type: GetFileType(path), Size: info.Size(), Modified: info.ModTime(), LastChecked: time.Now()}
			fmt.Println(foundFile)
			foundFile.Add(db)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
