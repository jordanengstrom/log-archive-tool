package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultArchiveDirName = "archives"
	historyFileName       = "archive_history.log"
	timeFormatFilename    = "20060102_150405"
	timeFormatHuman       = "2006-01-02 15:04:05 MST"
)

func main() {
	destFlag := flag.String("dest", "", "optional destination directory (default: <log-dir>/archives)")
	verbose := flag.Bool("v", false, "enable verbose logging")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <log-directory>\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	srcDir := flag.Arg(0)
	if !filepath.IsAbs(srcDir) {
		abs, err := filepath.Abs(srcDir)
		if err == nil {
			srcDir = abs
		}
	}

	info, err := os.Stat(srcDir)
	if err != nil {
		log.Fatalf("cannot access directory %q: %v", srcDir, err)
	}
	if !info.IsDir() {
		log.Fatalf("%q is not a directory", srcDir)
	}

	destDir := *destFlag
	if destDir == "" {
		destDir = filepath.Join(srcDir, defaultArchiveDirName)
	}
	if !filepath.IsAbs(destDir) {
		abs, err := filepath.Abs(destDir)
		if err == nil {
			destDir = abs
		}
	}

	if *verbose {
		log.Printf("source directory: %s\n", srcDir)
		log.Printf("destination directory: %s\n", destDir)
	}

	archivePath, filesArchived, totalBytes, err := createArchive(srcDir, destDir, *verbose)
	if err != nil {
		log.Fatalf("failed to create archive: %v", err)
	}

	if *verbose {
		log.Printf("archive created: %s", archivePath)
		log.Printf("files archived: %d, total bytes: %d", filesArchived, totalBytes)
	}

	if err := appendHistory(destDir, archivePath, filesArchived, totalBytes); err != nil {
		log.Printf("warning: failed to append to history file: %v", err)
	}

	fmt.Printf("Archive complete: %s\n", archivePath)
}

// createArchive collects regular files in srcDir (non-recursive) and writes them into a tar.gz
// placed in destDir. It skips files already in destDir and files that look compressed (.gz, .tgz, .tar.gz).
func createArchive(srcDir, destDir string, verbose bool) (archivePath string, filesArchived int, totalBytes int64, retErr error) {
	// ensure destDir exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		retErr = fmt.Errorf("cannot create destination dir %s: %w", destDir, err)
		return
	}

	now := time.Now()
	ts := now.Format(timeFormatFilename)
	archiveName := fmt.Sprintf("logs_archive_%s.tar.gz", ts)
	archivePath = filepath.Join(destDir, archiveName)

	tmpArchivePath := archivePath + ".tmp"

	outFile, err := os.Create(tmpArchivePath)
	if err != nil {
		retErr = fmt.Errorf("failed to create archive file: %w", err)
		return
	}

	// ensure temporary file is removed on error
	defer func() {
		if retErr != nil {
			if cerr := outFile.Close(); cerr != nil {
				log.Printf("warning: error closing tmp archive file: %v", cerr)
			}
			if remErr := os.Remove(tmpArchivePath); remErr != nil {
				log.Printf("warning: failed to remove tmp archive %s: %v", tmpArchivePath, remErr)
			}
		}
	}()

	gw := gzip.NewWriter(outFile)
	tw := tar.NewWriter(gw)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		retErr = fmt.Errorf("failed to list directory %s: %w", srcDir, err)
		return
	}

	for _, entry := range entries {
		// skip directories
		if entry.IsDir() {
			// skip the destination archive directory if it's inside srcDir
			if entry.Name() == filepath.Base(destDir) {
				if verbose {
					log.Printf("skipping archive directory: %s", entry.Name())
				}
				continue
			}
			// skip other directories in this simple tool
			continue
		}

		name := entry.Name()
		// skip compressed files
		lname := strings.ToLower(name)
		if strings.HasSuffix(lname, ".gz") || strings.HasSuffix(lname, ".tgz") || strings.HasSuffix(lname, ".tar.gz") {
			if verbose {
				log.Printf("skipping already compressed file: %s", name)
			}
			continue
		}
		// skip the history file if present in same dir (defensive)
		if name == historyFileName {
			if verbose {
				log.Printf("skipping history file: %s", name)
			}
			continue
		}

		fullPath := filepath.Join(srcDir, name)
		info, err := os.Lstat(fullPath)
		if err != nil {
			log.Printf("warning: unable to stat %s: %v", fullPath, err)
			continue
		}
		if !info.Mode().IsRegular() {
			if verbose {
				log.Printf("skipping non-regular file: %s", name)
			}
			continue
		}

		// open file
		f, err := os.Open(fullPath)
		if err != nil {
			log.Printf("warning: unable to open %s: %v", fullPath, err)
			continue
		}

		// prepare header
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			if cerr := f.Close(); cerr != nil {
				log.Printf("warning: error closing %s: %v", fullPath, cerr)
			}
			log.Printf("warning: unable to create tar header for %s: %v", fullPath, err)
			continue
		}
		// store only the filename (not full path) to keep archive tidy
		hdr.Name = name

		if err := tw.WriteHeader(hdr); err != nil {
			if cerr := f.Close(); cerr != nil {
				log.Printf("warning: error closing %s: %v", fullPath, cerr)
			}
			log.Printf("warning: unable to write tar header for %s: %v", fullPath, err)
			continue
		}

		n, err := io.Copy(tw, f)
		// ensure file is closed and note any close error
		if cerr := f.Close(); cerr != nil {
			log.Printf("warning: error closing %s: %v", fullPath, cerr)
		}
		if err != nil {
			log.Printf("warning: error writing file %s into archive: %v", fullPath, err)
			continue
		}

		filesArchived++
		totalBytes += n

		if verbose {
			log.Printf("archived %s (%d bytes)", name, n)
		}
	}

	// close writers to flush and check errors
	if cerr := tw.Close(); cerr != nil {
		retErr = fmt.Errorf("error closing tar writer: %w", cerr)
		return
	}
	if cerr := gw.Close(); cerr != nil {
		retErr = fmt.Errorf("error closing gzip writer: %w", cerr)
		return
	}
	if cerr := outFile.Close(); cerr != nil {
		retErr = fmt.Errorf("error closing output file: %w", cerr)
		return
	}

	// atomically rename tmp to final
	if err := os.Rename(tmpArchivePath, archivePath); err != nil {
		retErr = fmt.Errorf("failed to rename archive to final path: %w", err)
		return
	}

	return
}

// appendHistory writes an entry into a history file in destDir to record the archive action.
func appendHistory(destDir, archivePath string, files int, totalBytes int64) error {
	historyPath := filepath.Join(destDir, historyFileName)
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}

	line := fmt.Sprintf("%s\tarchive=%s\tfiles=%d\ttotal_bytes=%d\n", time.Now().Format(timeFormatHuman), filepath.Base(archivePath), files, totalBytes)
	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return fmt.Errorf("write history: %w", err)
	}
	if cerr := f.Close(); cerr != nil {
		return fmt.Errorf("close history file: %w", cerr)
	}
	return nil
}
