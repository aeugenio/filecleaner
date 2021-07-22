package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

/*
i made some assumptions about what kind of program would be needed by the vague description ahead of time.  initially,
i used go 1.16 and you can see this at https://github.com/aeugenio/filecleaner/blob/main/main.go  then, i was reading
about the qualified ide's environment and it said it used go 1.12, so i created
https://github.com/aeugenio/filecleaner/blob/go-1.12/main.go  which uses the older filepath.Walk function.

also, i'll call out that qualified said we'd only have access to the standard go libs, so i didnt use zerolog or viper
for logs and cli parsing.  that's what i've used in previous projects.  i'd much prefer to have log.Debug() and log.Info()
all over rather than Log.Println()
*/

func init() {
	var err error

	log.Print("initializing")
	flag.StringVar(&flagPath, "path", "", "the path of the directory to prune files from")
	flag.StringVar(&flagLogPath, "logPath", "", "the path to write filecleaner.log to; if not specified, logs are written to stdout")
	flag.BoolVar(&flagDryRun, "dryRun", false, "true to enable dry-run mode which only displays files that will be deleted but does not delete them")
	flag.StringVar(&flagDuration, "duration", "90", "the duration of time in days to evaluate the last modDate by. Calculated as now()-duration")
	flag.StringVar(&flagExactTime, "exactTime", "", "the time to use in identifying files to delete.  any files with a modTime before this value will be deleted.")
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

		log.Println("path=", path)
		log.Println("info=", info.Name())

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
			tokens := strings.Split(info.Name(), "-")
			mod := 0

			if len(tokens) == 4 {
				mod = -1
			} else if len(tokens) == 6 {
				mod = +1
			}

			year, e := strconv.Atoi(tokens[1+mod])
			if e != nil {
				log.Println("error with year in file ", path+"/"+info.Name())
				return e
			}

			month, e := strconv.Atoi(tokens[2+mod])
			if e != nil {
				log.Println("error with month in file ", path+"/"+info.Name())
				return e
			}

			day, e := strconv.Atoi(tokens[3+mod])
			if e != nil {
				log.Println("error with day in file ", path+"/"+info.Name())
				return e
			}

			fileTime := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

			if !info.IsDir() {
				isBefore := false
				if !exactTime.IsZero() {
					if fileTime.Before(exactTime) {
						isBefore = true
					}
				} else {
					if fileTime.Before(time.Now().Add(-time.Hour * 24 * 90)) {
						isBefore = true
					}
				}

				if isBefore {
					paths = append(paths, p)
					infos = append(infos, info)
				}
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
