package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"

	"mhx.at/gitlab/landscape/metrics-sender/pkg/config"
	"mhx.at/gitlab/landscape/metrics-sender/pkg/influx"
	"mhx.at/gitlab/landscape/metrics-sender/pkg/parser"

	"github.com/sirupsen/logrus"
)

var (
	version    = "0.0.0-src"
	configFile = flag.String("config", "config.yml", "Config file location")
)

func main() {
	var log = logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.TraceLevel) // is overwritten by configuration below

	log.Infof("TSA metrics-sender (Version: %s)", version)

	flag.Parse()

	log.Infof("Loading config from file: %s", *configFile)
	cfg, err := config.LoadConfig(*configFile)
	failOnError(err, fmt.Sprintf("Error opening config file: %s", *configFile), log)

	parsedLogLevel, err := logrus.ParseLevel(cfg.LogLevel)
	failOnError(err, fmt.Sprintf("Error parsing loglevel in config file: %s", cfg.LogLevel), log)
	log.SetLevel(parsedLogLevel)

	if cfg.LogFile != "" {
		logfile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_WRONLY, 0755)
		failOnError(err, fmt.Sprintf("Error opening log file: %s", cfg.LogFile), log)
		log.SetOutput(logfile)
		log.Infof("Writing to log file %s", cfg.LogFile)
	}

	// read files in specified source folder, sort them by modification time so older files are processed first
	files, err := os.ReadDir(cfg.SourceFolder)
	failOnError(err, "Could not read source folder", log)
	sort.Slice(files, func(i, j int) bool {
		ii, err := files[i].Info()
		failOnError(err, fmt.Sprintf("Could not get info of file %s", files[i].Name()), log)
		ij, err := files[j].Info()
		failOnError(err, fmt.Sprintf("Could not get info of file %s", files[j].Name()), log)
		return ii.ModTime().Before(ij.ModTime())
	})

	if len(files) <= 0 {
		log.Trace("No files to process, exiting")
		return
	}

	influxConnection, err := influx.CreateInfluxConnection(cfg.Influx)
	failOnError(err, "Could not connect to influx", log)

	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			log.Errorf("Could not get info of file %s: %v", file.Name(), err)
			continue
		}
		if !fileInfo.IsDir() {
			fullPath := path.Join(cfg.SourceFolder, file.Name())

			lines, err := readLines(fullPath)
			if err != nil {
				log.Errorf("Could not read file %s: %v", file.Name(), err)
				continue
			}

			pointsInFile, err := parser.Parse(lines)
			if err != nil {
				log.Errorf("Could not parse file %s: %v", file.Name(), err)
				continue
			}

			// for _, pointInFile := range pointsInFile {
			// 	fmt.Println(pointInFile.String())
			// }

			err = influx.Send(pointsInFile, influxConnection, cfg.Influx)
			if err != nil {
				log.Errorf("Could not send points of file %s to influx: %v", file.Name(), err)
				continue
			}

			err = os.Remove(fullPath)
			if err != nil {
				log.Errorf("Could not delete file %s", file.Name(), err)
				continue
			}

			log.Tracef("Successfully processed and sent metrics of file %s", file.Name())
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func failOnError(err error, msg string, log *logrus.Logger) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
