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
	"sort"
	"time"

	"github.com/roh/fileinventory/models"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	indexSource := indexCmd.String("source", "", "")
	indexLabel := indexCmd.String("label", "", "")
	indexCategory := indexCmd.String("category", "", "")
	indexDryrun := indexCmd.Bool("dryrun", false, "dryrun")
	indexSkipDiscovered := indexCmd.Bool("skip-discovered", false, "skip discovered files (path, source, and size already match")

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) < 2 {
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
	if *indexDryrun {
		fmt.Println("Running in dryrun mode")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite3", filepath.Join(homeDir, "index.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	models.CreateFoundFileTable(db)

	switch os.Args[1] {
	case "index":
		indexCmd.Parse(os.Args[2:])
		if *indexSource == "" {
			log.Fatal("Please specify a source flag, i.e. -source mylaptop")
		}
		scanPath(db, *indexSource, path, *indexCategory, *indexLabel, *indexSkipDiscovered, *indexDryrun)
	default:
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
}

func scanPath(db *sql.DB, source string, path string, category string, label string, skipDiscovered bool, dryrun bool) {
	foundFiles := walkFiles(path, source)
	fmt.Println()
	if len(foundFiles) == 0 {
		fmt.Println("No files found")
		return
	}
	prev, new, skipped := 0, 0, 0
	var foundFiles2 []models.FoundFile
	if skipDiscovered {
		for _, ff := range foundFiles {
			previousFF := models.GetFoundFileWithSize(db, source, ff.Path, ff.Size)
			if previousFF != nil {
				skipped++
			} else {
				foundFiles2 = append(foundFiles2, ff)
			}
		}
	}
	if foundFiles2 == nil {
		fmt.Println("No new files found")
		return
	}
	displayFoundFilesSummary(foundFiles2)
	if dryrun {
		return
	}
	fmt.Println("\nCalculating md5 sums and adding to database:")
	for _, ff := range foundFiles2 {
		md5hash := getMd5hash(ff.Path)
		previousFF := models.GetFoundFileWithMd5hash(db, source, ff.Path, md5hash)
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
	fmt.Println("\nNew:", new, "   Previous:", prev, "   Skipped:", skipped)
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
	sort.SliceStable(foundFiles, func(i, j int) bool {
		p1, p2 := foundFiles[i].Path, foundFiles[j].Path
		d1, d2 := filepath.Dir(p1), filepath.Dir(p2)
		return d1 < d2
	})
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
	fmt.Print("Discovered          Modified            Size (KB)    Type        Name\n")
	for _, ff := range foundFiles {
		s := float32(ff.Size) / 1000
		fmt.Printf("%s    %s    %9.f    %-8s    %s\n", ff.Discovered.Format("2006-01-02 15:04"), ff.Modified.Format("2006-01-02 15:04"), s, ff.Type, ff.Name)
	}
}
