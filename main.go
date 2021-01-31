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
	"strings"
	"time"

	"github.com/roh/fileinventory/inventory"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) < 2 {
		fmt.Println("expected 'index', 'ls', or 'health' command")
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

		inventory.Init(*dbPath)
		defer inventory.Close()
		if *source == "" {
			log.Fatal("Please specify a source flag, i.e. -source mylaptop")
		}
		indexPath(*source, path, *category, *subcategory, *label, *tags, *indexReindexDiscovered)
	case "ls":
		lsCmd := flag.NewFlagSet("ls", flag.ExitOnError)
		source := lsCmd.String("source", "", "")
		dbPath := lsCmd.String("db", "", "database path - defaults to $HOMEDIR/index.db")
		new := lsCmd.Bool("new", false, "")
		lsCmd.Parse(os.Args[2:])

		inventory.Init(*dbPath)
		defer inventory.Close()
		if *new {
			checkNewFiles(*source, path)
		} else {
			listFiles(*source, path)
		}
	case "health":
		healthCmd := flag.NewFlagSet("health", flag.ExitOnError)
		source := healthCmd.String("source", "", "")
		dbPath := healthCmd.String("db", "", "database path - defaults to $HOMEDIR/index.db")
		healthCmd.Parse(os.Args[2:])

		inventory.Init(*dbPath)
		defer inventory.Close()
		checkHealthFiles(*source, path)
	default:
		fmt.Println("expected 'index' subcommand")
		os.Exit(1)
	}
}

func checkHealthFiles(source string, path string) {
	foundFiles := walkFiles(path, source)
	var notFoundFiles []inventory.FoundFile
	fmt.Println()
	nFound := 0
	nNotFound := 0
	nNotIndexed := 0
	for _, ff := range foundFiles {
		previousFF := inventory.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
		if previousFF == nil {
			nNotIndexed++
			continue
		}
		otherFFs := inventory.GetFoundFileOtherSourcesWithMd5hash(source, previousFF.Md5hash)
		if len(otherFFs) == 0 {
			notFoundFiles = append(notFoundFiles, ff)
			nNotFound++
			continue
		}
		nFound++
		fmt.Println(ff.Path)
		for _, off := range otherFFs {
			fmt.Printf("%-16s    %s    %s\n", off.Source, off.LastChecked.Format("2006-01-02 15:04"), off.Path)
		}
		fmt.Println()
	}
	fmt.Println()
	if len(notFoundFiles) > 0 {
		fmt.Println("Files without other sources:")
		for _, ff := range notFoundFiles {
			fmt.Println(ff.Path)
		}
		fmt.Println()
	}

	if nNotIndexed > 0 {
		fmt.Println(nNotIndexed, "files are not indexed")
	}
	if nFound+nNotFound > 0 {
		fmt.Printf("Found %d out of %d files. Health is %.1f%%\n", nFound, nFound+nNotFound, float32(nFound)/(float32(nFound+nNotFound))*100)
	}
}

// Searches for files with same filesize and modified timestamp
func checkNewFiles(source string, path string) {
	foundFiles := walkFiles(path, source)
	var notFoundFiles []inventory.FoundFile
	fmt.Println()
	nNotFound := 0
	nNotIndexed := 0
	for _, ff := range foundFiles {
		previousFF := inventory.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
		if previousFF != nil {
			ff.Discovered = previousFF.Discovered
			otherFFs := inventory.GetFoundFileOtherSourcesWithMd5hash(source, previousFF.Md5hash)
			if len(otherFFs) == 0 {
				notFoundFiles = append(notFoundFiles, ff)
				nNotFound++
			}
		} else {
			ff.Discovered = time.Time{}
			nNotIndexed++
			similarFiles := inventory.GetSimilarFoundFileSourcesWithSizeAndModified(ff.Size, ff.Modified)
			if len(similarFiles) == 0 {
				notFoundFiles = append(notFoundFiles, ff)
				nNotFound++
			} else {
				fmt.Println(ff.Path)
				for _, f := range similarFiles {
					fmt.Printf("%-16s    %s    %s\n", f.Source, f.LastChecked.Format("2006-01-02 15:04"), f.Path)
				}
			}
		}
	}
	if nNotFound > 0 {
		fmt.Printf("\n%s\n\n", strings.Repeat("-", 80))
		fmt.Println(nNotFound, "files do not have any similar/redundant files in other sources:")
		for _, f := range notFoundFiles {
			fmt.Println(f.Path)
		}
	}
}

func listFiles(source string, path string) {
	foundFiles := walkFiles(path, source)
	fmt.Println()
	var foundFiles2 []inventory.FoundFile
	for _, ff := range foundFiles {
		previousFF := inventory.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
		if previousFF == nil {
			ff.Discovered = time.Time{}
		} else {
			ff.Discovered = previousFF.Discovered
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
	if !reindexDiscovered {
		var foundFiles2 []inventory.FoundFile
		for _, ff := range foundFiles {
			previousFF := inventory.GetFoundFileWithSizeAndModified(source, ff.Path, ff.Size, ff.Modified)
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
	unit, unitName := bestUnit(int64(sizeTotal))
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
		fmt.Printf("\nProgress  %.1f%%  %.f/%.f %-40s", (sizeSkipped+sizeProcessed)/sizeTotal*100, (sizeSkipped+sizeProcessed)/unit, sizeTotal/unit, unitName)
		speedFmt := ""
		if speed > 0 {
			// TODO: Use a window to get a more accurate estimate
			speedFmt = fmt.Sprintf("Speed: %.1f MB/s Remaining %.fs", speed*0.000001, remaining)
		}
		fmt.Printf("\nTime elapsed: %s %s", time.Since(start), speedFmt)
		md5hash := getMd5hash(ff.Path)
		previousFF := inventory.GetFoundFileWithMd5hash(source, ff.Path, md5hash)
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

func walkFiles(path string, source string) []inventory.FoundFile {
	var foundFiles []inventory.FoundFile
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
			ff := inventory.FoundFile{Source: source, Path: path}
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

func displayFoundFilesSummary(foundFiles []inventory.FoundFile) {
	var dir string
	lastDir := filepath.Dir(foundFiles[0].Path)
	var dirFiles []inventory.FoundFile
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

func displayFoundFileList(foundFiles []inventory.FoundFile) {
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
