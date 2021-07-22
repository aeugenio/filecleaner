package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"
)

var flagPath string
var flagDuration string
var flagExactTime string
var flagDryRun bool
var flagLogPath string

var duration time.Duration
var exactTime time.Time
var logFile *os.File

func init() {
	var err error

	log.Print("initializing")
	flag.StringVar(&flagPath, "path", "", "the path of the directory to prune files from")
	flag.StringVar(&flagLogPath, "logPath", "", "the path to write filecleaner.log to; if not specified, logs are written to stdout")
	flag.BoolVar(&flagDryRun, "dryRun", false, "true to enable dry-run mode which only displays files that will be deleted but does not delete them")
	flag.StringVar(&flagDuration, "duration", "", "the duration of time to evaluate the last modDate by. Calculated as now()-duration.  A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, such as \"300ms\", \"-1.5h\" or \"2h45m\". Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\".")
	flag.StringVar(&flagExactTime, "exactTime", "", "the time to use in identifying files to delete.  any files with a modTime before this value will be deleted.  setting this flag causes the program to use it and not duration regardless of duration being set or not")
	flag.Parse()

	log.Println("flagPath =", flagPath)
	log.Println("flagDuration =", flagDuration)
	log.Println("flagExactTime =", flagExactTime)
	log.Println("flagDryRun =", flagDryRun)
	log.Println("flagLogPath =", flagLogPath)

	if flagPath == "" {
		log.Fatal("missing required param: flagPath")
	}

	if flagDuration == "" && flagExactTime == "" {
		log.Fatal("a duration or exactTime must be specified")
	}

	if flagDuration != "" && flagExactTime != "" {
		log.Fatal("only duration or exactTime can be specified")
	}

	if flagDuration != "" {
		localDuration, err := time.ParseDuration(flagDuration)
		if err != nil {
			log.Fatal("invalid duration flag", err.Error())
		}
		duration = localDuration
	}

	//if exactTime is passed in, parse  it
	if flagExactTime != "" {
		layout := "2006-01-02T15:04:05.000Z"
		exactTime, err = time.Parse(layout, flagExactTime)
		if err != nil {
			log.Fatal("invalid exactTime flag", err.Error())
		}
	}

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

func findFilesToDelete(path string) (paths []string, infos []os.FileInfo, err error) {
	log.Println("about to findFilesToDelete")
	err = filepath.Walk(path, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if !info.IsDir() {
			isBefore := false
			if !exactTime.IsZero() {
				if info.ModTime().Before(exactTime) {
					isBefore = true
				}
			} else {
				if info.ModTime().Before(time.Now().Add(-duration)) {
					isBefore = true
				}
			}

			if isBefore {
				paths = append(paths, p)
				infos = append(infos, info)
			}
		}

		return nil
	})

	log.Println("found", len(paths), "files to delete")

	return
}

func dryRun(paths []string, fileInfos []os.FileInfo) {
	log.Println("dryRun is enabled so not deleting any files.  These files would be deleted if dryRun==false")
	for i, path := range paths {
		log.Println(path, "with last modDate of ", fileInfos[i].ModTime())
	}
}

func deleteFiles(paths []string, fileInfos []os.FileInfo) {
	log.Println("dryRun is not enabled")

	for i, path := range paths {
		log.Println("removing file ", path, "with last modDate of ", fileInfos[i].ModTime())

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

	paths, fileInfos, err := findFilesToDelete(flagPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(paths) > 0 {
		if flagDryRun {
			dryRun(paths, fileInfos)
		} else {
			deleteFiles(paths, fileInfos)
		}
	}
	log.Println("filecleaner done")
}
