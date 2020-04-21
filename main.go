package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

func main() {
	var (
		logLevel   = kingpin.Flag("loglevel", "Log level: [debug, info, warn, error, fatal]").Short('l').Default("info").String()
		timeout    = kingpin.Flag("timeout", "Timeout: overall timeout for all collectors").Short('t').Default("0s").Duration()
		config     = kingpin.Flag("config", "Path to collectors configuration file").Short('c').Required().String()
		baseDir    = kingpin.Flag("basedir", "Temporary base directory to create the resulting collection tarball").Short('b').Default("/tmp").String()
		resultsDir = kingpin.Flag("results-dir", "Directory to store the resulting collection tarball").Short('r').Default(".").String()
	)

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	parsedLogLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Cannot set log level, error: %s", err.Error())
		os.Exit(-1)
	}

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true

	log.SetFormatter(customFormatter)
	log.SetLevel(parsedLogLevel)
	log.SetOutput(os.Stdout)

	scheduler, err := NewScheduler(*config, timeout, *baseDir, *resultsDir)
	if err != nil {
		log.Errorf("Cannot enable scheduler, exiting, error: %s", err.Error())
		os.Exit(-1)
	}

	log.Fatal(scheduler.Start())
}
