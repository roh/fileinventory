package main

import (
	"crypto/md5"
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
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) < 2 {
		fmt.Println("expected 'index' or 'show' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		source := indexCmd.String("source", "", "")
		label := indexCmd.String("label", "", "")
		category := indexCmd.String("category", "", "")
		subcategory := indexCmd.String("subcategory", "", "")
		tags := indexCmd.String("tags", "", "")
		dbPath := indexCmd.String("db", "", "database path - defaults to $HOMEDIR/index.db")
		indexReindexDiscovered := indexCmd.Bool("reindex", false, "reindex previously discovered files that haven't changed")
		indexCmd.Parse(os.Args[2:])

		models.InitDB(*dbPath)
		defer models.CloseDB()
		if *source == "" {
			log.Fatal("Please specify a source flag, i.e. -source mylaptop")
		}
		indexPath(*source, path, *category, *subcategory, *label, *tags, *indexReindexDiscovered)
	case "show":
		showCmd := flag.NewFlagSet("show", flag.ExitOnError)
		source := showCmd.String("source", "", "")
		dbPath := showCmd.String("db", "", "database path - defaults to $HOMEDIR/index.db")
		showCmd.Parse(os.Args[2:])

		models.InitDB(*dbPath)
		defer models.CloseDB()
		showFiles(*source, path)
	default:
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
}

func showFiles(source string, path string) {
	foundFiles := walkFiles(path, source)
	fmt.Println()
	var foundFiles2 []models.FoundFile
	for _, ff := range foundFiles {
		previousFF := models.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
		if previousFF == nil {
			ff.Discovered = time.Time{}
		}
		foundFiles2 = append(foundFiles2, ff)
	}
	if foundFiles2 == nil {
		fmt.Println("No new files found")
		return
	}
	displayFoundFilesSummary(foundFiles2)
}

func indexPath(source string, path string, category string, subcategory string, label string, tags string, reindexDiscovered bool) {
	foundFiles := walkFiles(path, source)
	fmt.Println()
	if len(foundFiles) == 0 {
		fmt.Println("No files found")
		return
	}
	numSkipped, numProcessed, numTotal := 0, 0, len(foundFiles)
	var sizeSkipped, sizeProcessed, sizeTotal float32
	for _, ff := range foundFiles {
		sizeTotal += float32(ff.Size)
	}
	var unit float32
	var unitName string
	switch {
	case sizeTotal > 100*1000*1000*1000:
		unit = 1000 * 1000 * 1000
		unitName = "GB"
	case sizeTotal > 100*1000*1000:
		unit = 1000 * 1000
		unitName = "MB"
	case sizeTotal > 100*1000:
		unit = 1000
		unitName = "KB"
	default:
		unit = 1
		unitName = "bytes"
	}
	if !reindexDiscovered {
		var foundFiles2 []models.FoundFile
		for _, ff := range foundFiles {
			previousFF := models.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
			if previousFF != nil {
				numSkipped++
				sizeSkipped += float32(ff.Size)
			} else {
				foundFiles2 = append(foundFiles2, ff)
			}
		}
		if foundFiles2 == nil {
			fmt.Println("No new files found")
			return
		}
		foundFiles = foundFiles2
	}
	fmt.Println("\nCalculating md5 sums and adding to database...")
	if numSkipped == 1 {
		fmt.Printf("\nSkipping 1 file, size %.f %s\n", sizeSkipped, unitName)
	} else if numSkipped >= 2 {
		fmt.Printf("\nSkipping %d files, size %.f %s\n", numSkipped, sizeSkipped/unit, unitName)
	}
	prev, new := 0, 0
	start := time.Now()
	for _, ff := range foundFiles {
		// sizeRemaining := sizeTotal - sizeSkipped - sizeProcessed
		timeElapsed := time.Since(start).Seconds()
		speed := float32(0)
		remaining := float32(0)
		if timeElapsed > 0 && numProcessed > 0 && sizeProcessed > 1000 {
			speed = sizeProcessed / float32(timeElapsed)
			remaining = (sizeTotal - sizeSkipped - sizeProcessed) / speed
		}
		l := fmt.Sprintf("(%1d/%1d): %s", numProcessed+1, numTotal-numSkipped, ff.Name)
		fmt.Printf("\n%-80s", l)
		fmt.Printf("\n%-80s", "")
		fmt.Printf("\nProgress  %.1f%%  %.f/%.f %-40s", sizeProcessed/sizeTotal*100, sizeProcessed/unit, sizeTotal/unit, unitName)
		speedFmt := ""
		if speed > 0 {
			// TODO: Use a window to get a more accurate estimate
			speedFmt = fmt.Sprintf("Speed: %.1f MB/s Remaining %.fs", speed*0.000001, remaining)
		}
		fmt.Printf("\nTime elapsed: %s %s", time.Since(start), speedFmt)
		md5hash := getMd5hash(ff.Path)
		previousFF := models.GetFoundFileWithMd5hash(source, ff.Path, md5hash)
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
			if len(subcategory) > 0 {
				ff.Subcategory = subcategory
			}
		}
		if len(label) > 0 {
			ff.Label = label
		}
		if len(tags) > 0 {
			ff.Tags = tags
		}
		ff.LastChecked = time.Now()
		ff.Save()
		numProcessed++
		sizeProcessed += float32(ff.Size)
		fmt.Print("\u001b[1000D\u001b[3A")
	}
	fmt.Printf("\n%-80.80s", "")
	l := fmt.Sprintf("Complete!")
	fmt.Printf("\n%-80.80s\n", l)
	l = fmt.Sprintf("Processed %d new and %d previous files", new, prev)
	fmt.Printf("%-80.80s\n", l)
	fmt.Printf("Processed size: %.f %s\n", sizeTotal/unit, unitName)
}

func walkFiles(path string, source string) []models.FoundFile {
	var foundFiles []models.FoundFile
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if IsHidden(info.Name()) {
				fmt.Println("Skipping folder", info.Name())
				return filepath.SkipDir
			}
			fmt.Printf("Scanning folder %s\n", info.Name())
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
	var dir string
	lastDir := filepath.Dir(foundFiles[0].Path)
	var dirFiles []models.FoundFile
	var sizeTotal int64
	for _, ff := range foundFiles {
		sizeTotal += ff.Size
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
	fmt.Println("\nFound", len(foundFiles), "files")
	fmt.Printf("Total Size: %d\n", sizeTotal)
}

func displayFoundFileList(foundFiles []models.FoundFile) {
	fmt.Print("Discovered          Modified            Size (KB)    Type        Name\n")
	for _, ff := range foundFiles {
		s := float32(ff.Size) / 1000
		d := ""
		if !ff.Discovered.IsZero() {
			d = ff.Discovered.Format("2006-01-02 15:04")
		}
		fmt.Printf("%16s    %s    %9.f    %-8s    %s\n", d, ff.Modified.Format("2006-01-02 15:04"), s, ff.Type, ff.Name)
	}
}
