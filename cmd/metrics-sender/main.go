package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"sort"
	"syscall"
	"time"

	"mhx.at/gitlab/landscape/metrics-sender/pkg/config"
	"mhx.at/gitlab/landscape/metrics-sender/pkg/influx"
	"mhx.at/gitlab/landscape/metrics-sender/pkg/parser"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/remeh/sizedwaitgroup"
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
		logfile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
		failOnError(err, fmt.Sprintf("Error opening log file: %s", cfg.LogFile), log)
		log.SetOutput(logfile)
		log.Infof("Writing to log file %s", cfg.LogFile)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signalChan:
			log.Infof("Got SIGINT/SIGTERM, exiting.")
			cancel()
			os.Exit(0)
		case <-ctx.Done():
			log.Infof("Exiting.")
			os.Exit(0)
		}
	}()

	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	run(ctx, cfg, log)
}

func run(ctx context.Context, cfg *config.Configuration, log *logrus.Logger) error {

	process(cfg, cfg.RereadFolderSeconds*time.Second, log) // initial processing, because first tick only happens after interval

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.Tick(cfg.ProcessIntervalSeconds * time.Second):
			process(cfg, cfg.RereadFolderSeconds*time.Second, log)
		}
	}
}

func process(cfg *config.Configuration, rereadFolderInterval time.Duration, log *logrus.Logger) {

	// process until done
	for done := false; !done; {
		done = processWithTimeout(cfg, rereadFolderInterval, log)
	}
}

func processWithTimeout(cfg *config.Configuration, timeout time.Duration, log *logrus.Logger) bool {
	// read files in specified source folder, sort them by modification time so newer files are processed first
	files, err := os.ReadDir(cfg.SourceFolder)
	if err != nil {
		log.Errorf("Could not read source folder: %v", err)
		return true
	}
	if len(files) <= 0 {
		log.Info("No files to process")
		return true
	}

	// get modification times
	modTimes := make([]time.Time, len(files))
	for i := range modTimes {
		ii, err := files[i].Info()
		if err != nil {
			log.Errorf("Could not read info of file %s: %v", files[i].Name(), err)
			return true
		}
		modTimes[i] = ii.ModTime()
	}

	// sort files by modification time
	// to make them process latest first
	sort.Slice(files, func(i, j int) bool {
		return modTimes[i].After(modTimes[j])
	})

	influxConnection, err := influx.CreateInfluxConnection(cfg.Influx)
	if err != nil {
		log.Errorf("Could not connect to influx: %v", err)
		return true
	}
	defer influxConnection.Close()

	startTime := time.Now()

	swg := sizedwaitgroup.New(cfg.MaxConcurrentWorkers)

	log.Tracef("Starting processing of %d files", len(files))

	earlyReturn := false
	for _, file := range files {

		if time.Now().Sub(startTime) > timeout {
			// already taking longer than timeout -> return early
			log.Warnf("Processing of directory took longer than %.0f seconds: re-starting...", timeout.Seconds())
			earlyReturn = true
			break
		}

		swg.Add() // blocks if maximum number of workers reached, until a worker is finished
		go func(file fs.DirEntry) {
			defer swg.Done()
			processSingleFile(file, influxConnection, cfg, log)
		}(file)
	}

	swg.Wait()

	if !earlyReturn {
		log.Tracef("Finished processing of %d files", len(files))
	}

	return !earlyReturn
}

func processSingleFile(file fs.DirEntry, influxConnection client.Client, cfg *config.Configuration, log *logrus.Logger) {
	fileInfo, err := file.Info()
	if err != nil {
		log.Errorf("Could not get info of file %s: %v", file.Name(), err)
		return
	}
	if !fileInfo.IsDir() {
		fullPath := path.Join(cfg.SourceFolder, file.Name())

		lines, err := readLines(fullPath)
		if err != nil {
			log.Errorf("Could not read file %s: %v", file.Name(), err)
			return
		}

		pointsInFile, err := parser.Parse(lines)
		if err != nil {
			log.Errorf("Could not parse file %s: %v", file.Name(), err)
			return
		}

		err = influx.Send(pointsInFile, influxConnection, cfg.Influx)
		if err != nil {
			log.Errorf("Could not send points of file %s to influx: %v", file.Name(), err)
			return
		}

		err = os.Remove(fullPath)
		if err != nil {
			log.Errorf("Could not delete file %s: %v", file.Name(), err)
			return
		}

		log.Tracef("Successfully processed and sent metrics of file %s", file.Name())
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

	// NOTE: some lines can get really long, that's why we allocate a long buffer to use instead of the default buffer
	// see MaxScanTokenSize and https://pkg.go.dev/bufio#NewScanner
	const maxCapacity = 512 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

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
