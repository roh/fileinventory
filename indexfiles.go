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

	root := "/Users/roh/Development/fileindexer"
	foundFiles := scanFiles(root, *sourceFlag)
	if len(foundFiles) == 0 {
		fmt.Println("No files found")
		return
	}
	displayFoundFilesSummary(foundFiles)
	fmt.Print("\nCalculating md5 sums and adding to database")
	for _, ff := range foundFiles {
		ff.Md5hash = getMd5hash(ff.Path)
		ff.LastChecked = time.Now()
		ff.Add(db)
		fmt.Print(".")
	}
	fmt.Println()
}

func scanFiles(root string, source string) []models.FoundFile {
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
			ff := models.FoundFile{Source: source, Path: path, Name: info.Name(), Extension: GetNormalizedExtension(path), Type: GetFileType(path), Size: info.Size(), Modified: info.ModTime(), Discovered: time.Now()}
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
	for _, ff := range foundFiles {
		dir = filepath.Dir(ff.Path)
		if dir != lastDir {
			println(dir)
			displayFoundFileList(dirFiles, nameLen)
			lastDir = dir
			nameLen = 4
			dirFiles = nil
		}
		dirFiles = append(dirFiles, ff)
		nameLen = Max(nameLen, len(ff.Name))

	}
	println(dir)
	displayFoundFileList(dirFiles, nameLen)
}

func displayFoundFileList(foundFiles []models.FoundFile, nameLen int) {
	fmt.Print("Name", strings.Repeat(" ", nameLen+1), "Type           Size\n")
	for _, ff := range foundFiles {
		ffNameLen := len(ff.Name)
		fmt.Print(ff.Name, strings.Repeat(" ", nameLen-ffNameLen))
		fmt.Printf("     %-10s     %d\n", ff.Type, ff.Size)
	}
}
