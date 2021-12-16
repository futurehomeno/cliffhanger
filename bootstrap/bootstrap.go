package bootstrap

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var workDir string

// GetWorkingDirectory returns a working directory passed through the -c option.
func GetWorkingDirectory() string {
	if workDir != "" {
		return workDir
	}

	flag.StringVar(&workDir, "c", "", "Working directory.")
	flag.Parse()

	if workDir == "" {
		workDir = "./"
	}

	return workDir
}

// InitializeLogger initializes logger with an optional log rotation.
func InitializeLogger(logFile string, level string, logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	if logFile != "" {
		l := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    5, // MiB
			MaxBackups: 2,
		}

		log.SetOutput(l)
	}
}

// WaitForShutdown blocks code execution until a shutdown signal occurs.
func WaitForShutdown() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	<-signals
}
