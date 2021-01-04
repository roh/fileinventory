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
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	indexSource := indexCmd.String("source", "", "")
	indexLabel := indexCmd.String("label", "", "")
	indexCategory := indexCmd.String("category", "", "")
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
		runIndexFiles(*indexSource, *indexCategory, *indexLabel)
	default:
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
}

func runIndexFiles(source string, category string, label string) {
	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	models.CreateFoundFileTable(db)

	root := "/Users/roh/Development/fileindexer"
	foundFiles := scanFiles(db, root, source)
	fmt.Println()
	if len(foundFiles) == 0 {
		fmt.Println("No files found")
		return
	}
	displayFoundFilesSummary(foundFiles)
	fmt.Println("\nCalculating md5 sums and adding to database:")
	for _, ff := range foundFiles {
		md5hash := getMd5hash(ff.Path)
		if ff.Md5hash != md5hash {
			// File is "new" if md5hash is different
			fmt.Println("\nWarning: md5 hash has changed for file", ff.Path)
			ff.Md5hash = md5hash
			ff.Discovered = time.Now()
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
	fmt.Println()
}

func scanFiles(db *sql.DB, root string, source string) []models.FoundFile {
	var foundFiles []models.FoundFile
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if IsHidden(info.Name()) {
				fmt.Println("Skipping folder", info.Name())
				return filepath.SkipDir
			}
			fmt.Printf("Scanning folder %s...\n", info.Name())
		} else if IsHidden(info.Name()) {
			return nil
		} else {
			var ff models.FoundFile
			previousFF := models.GetFoundFile(db, source, path)
			if previousFF != nil {
				ff = *previousFF
			} else {
				ff = models.FoundFile{Source: source, Path: path}
			}
			ff.Name = info.Name()
			ff.Extension = GetNormalizedExtension(path)
			ff.Type = GetFileType(path)
			ff.Size = info.Size()
			ff.Modified = info.ModTime()
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
	nameLen := 4
	typeLen := 4
	for _, ff := range foundFiles {
		dir = filepath.Dir(ff.Path)
		if dir != lastDir {
			println(dir)
			displayFoundFileList(dirFiles, nameLen, typeLen)
			lastDir = dir
			nameLen = 4
			typeLen = 4
			dirFiles = nil
		}
		dirFiles = append(dirFiles, ff)
		nameLen = Max(nameLen, len(ff.Name))
		typeLen = Max(typeLen, len(ff.Type))

	}
	println(dir)
	displayFoundFileList(dirFiles, nameLen, typeLen)
}

func displayFoundFileList(foundFiles []models.FoundFile, nameLen int, typeLen int) {
	fmt.Print("Name", strings.Repeat(" ", nameLen))
	fmt.Print("Type", strings.Repeat(" ", typeLen))
	fmt.Print("Size (MB)    Modified            Discovered\n")
	for _, ff := range foundFiles {
		s := float32(ff.Size) / 1000 / 1000
		fmt.Print(ff.Name, strings.Repeat(" ", nameLen-len(ff.Name)))
		fmt.Printf("    %s%s", ff.Type, strings.Repeat(" ", typeLen-len(ff.Type)))
		fmt.Printf("    %-9.2f    %s    %s\n", s, ff.Modified.Format("2006-01-02 15:04"), ff.Discovered.Format("2006-01-02 15:04"))
	}
}
