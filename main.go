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
	"time"

	"github.com/roh/fileinventory/models"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	indexSource := indexCmd.String("source", "", "")
	indexLabel := indexCmd.String("label", "", "")
	indexCategory := indexCmd.String("category", "", "")

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "index":
		indexCmd.Parse(os.Args[2:])
		if *indexSource == "" {
			log.Fatal("Please specify a source flag, i.e. -source mylaptop")
		}
		scanPath(*indexSource, path, *indexCategory, *indexLabel)
	default:
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
}

func scanPath(source string, path string, category string, label string) {
	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	models.CreateFoundFileTable(db)

	foundFiles := walkFiles(path, source)
	fmt.Println()
	if len(foundFiles) == 0 {
		fmt.Println("No files found")
		return
	}
	displayFoundFilesSummary(foundFiles)
	fmt.Println("\nCalculating md5 sums and adding to database:")
	prev := 0
	new := 0
	for _, ff := range foundFiles {
		md5hash := getMd5hash(ff.Path)
		previousFF := models.GetFoundFile(db, source, ff.Path, md5hash)
		if previousFF != nil {
			// File is "new" if md5hash is different
			previousFF.LastChecked = ff.LastChecked
			previousFF.Type = ff.Type
			previousFF.Size = ff.Size
			previousFF.Modified = ff.Modified
			ff = *previousFF
			prev++
			// TODO: Detect md5 hash changes and warn
		} else {
			new++
		}
		ff.Md5hash = md5hash
		if len(category) > 0 {
			ff.Category = category
		}
		if len(label) > 0 {
			ff.Label = label
		}
		ff.LastChecked = time.Now()
		ff.Save(db)
		fmt.Print(".")
	}
	fmt.Println("\nNew:", new, "   Previous:", prev)
}

func walkFiles(path string, source string) []models.FoundFile {
	var foundFiles []models.FoundFile
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if IsHidden(info.Name()) {
				fmt.Println("Skipping folder", info.Name())
				return filepath.SkipDir
			}
			fmt.Printf("Scanning folder %s...\n", info.Name())
		} else if IsHidden(info.Name()) {
			return nil
		} else {
			ff := models.FoundFile{Source: source, Path: path}
			ff.Name = info.Name()
			ff.Extension = GetNormalizedExtension(path)
			ff.Type = GetFileType(path)
			ff.Size = info.Size()
			ff.Modified = info.ModTime()
			ff.Discovered = time.Now()
			foundFiles = append(foundFiles, ff)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return foundFiles
}

func getMd5hash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func displayFoundFilesSummary(foundFiles []models.FoundFile) {
	fmt.Println("Found", len(foundFiles), "files")
	var dir string
	lastDir := filepath.Dir(foundFiles[0].Path)
	var dirFiles []models.FoundFile
	for _, ff := range foundFiles {
		dir = filepath.Dir(ff.Path)
		if dir != lastDir {
			fmt.Printf("\n%s\n", lastDir)
			displayFoundFileList(dirFiles)
			lastDir = dir
			dirFiles = nil
		}
		dirFiles = append(dirFiles, ff)

	}
	fmt.Printf("\n%s\n", dir)
	displayFoundFileList(dirFiles)
}

func displayFoundFileList(foundFiles []models.FoundFile) {
	fmt.Print("Discovered          Modified            Size (MB)    Type        Name\n")
	for _, ff := range foundFiles {
		s := float32(ff.Size) / 1000 / 1000
		fmt.Printf("%s    %s    %-5.f        %-8s    %s\n", ff.Discovered.Format("2006-01-02 15:04"), ff.Modified.Format("2006-01-02 15:04"), s, ff.Type, ff.Name)
	}
}
