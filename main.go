package main

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

var flagPath string
var flagDuration string
var flagDryRun bool
var flagLogPath string

var duration time.Duration
var logFile *os.File

func init() {
	log.Print("initializing")
	flag.StringVar(&flagPath, "path", "", "the path of the directory to prune files from")
	flag.StringVar(&flagLogPath, "logPath", "", "the path to write filecleaner.log to; if not specified, logs are written to stdout")
	flag.BoolVar(&flagDryRun, "dryRun", false, "true to enable dry-run mode which only displays files that will be deleted but does not delete them")
	flag.StringVar(&flagDuration, "duration", "5m", "the duration of time to evaluate the last modDate by. Calculated as now()-duration.  A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, such as \"300ms\", \"-1.5h\" or \"2h45m\". Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".")
	flag.Parse()

	if flagPath == "" {
		log.Fatal("missing required param: flagPath")
	}

	localDuration, err := time.ParseDuration(flagDuration)
	if err != nil {
		log.Fatal("invalid duration flag", err.Error())
	}

	log.Println("path =", flagPath)
	log.Println("duration =", flagDuration)
	log.Println("dryRun =", flagDryRun)
	log.Println("logPath =", flagLogPath)

	duration = localDuration

	log.Println("initializing logs")
	initLogs()
}

func initLogs() {
	var err error
	if flagLogPath != "" {
		logFile, err = os.OpenFile(flagLogPath+"/filecleaner.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("writing logs to", flagLogPath)
		log.SetOutput(logFile)
	} else {
		log.SetOutput(os.Stdout)
		log.Println("writing logs to STDOUT")
	}
}

func findFilesToDelete(path string) (paths []string, infos []fs.DirEntry, err error) {
	log.Println("about to findFilesToDelete")
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		info, _ := d.Info()
		if !d.IsDir() && info.ModTime().Before(time.Now().Add(-duration)) {
			paths = append(paths, p)
			infos = append(infos, d)
		}
		return nil
	})

	log.Println("found", len(paths), "files to delete")

	return
}

func dryRun(paths []string, dirEntries []fs.DirEntry) {
	log.Println("dryRun is enabled so not deleting any files.  These files would be deleted if dryRun==false")
	for i, info := range dirEntries {
		fileInfo, _ := info.Info()
		log.Println(paths[i], "with last modDate of ", fileInfo.ModTime())
	}
}

func deleteFiles(paths []string, dirEntries []fs.DirEntry) {
	log.Println("dryRun is not enabled")

	for i, path := range paths {
		fileInfo, _ := dirEntries[i].Info()
		log.Println("removing file ", paths[i], "with last modDate of ", fileInfo.ModTime())

		err := os.Remove(path)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func main() {
	//close the logfile when the program ends.  i would love to handle the possible error on closing, but doesnt seem
	//to be consensus on how to do that with a defer.  i found https://www.joeshaw.org/dont-defer-close-on-writable-files/
	//but not gonna overdo this simple program
	defer logFile.Close()

	log.Println("starting filecleaner")

	paths, dirEntries, err := findFilesToDelete(flagPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(paths) > 0 {
		if flagDryRun {
			dryRun(paths, dirEntries)
		} else {
			deleteFiles(paths, dirEntries)
		}
	}
	log.Println("filecleaner done")
}
