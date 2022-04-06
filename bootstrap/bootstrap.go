package bootstrap

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// GetConfigurationDirectory returns a configuration directory passed through the -c option.
func GetConfigurationDirectory() string {
	const c = "c"

	if flag.Lookup(c) == nil {
		flag.String(c, "", "Configuration directory.")
		flag.Parse()
	}

	return flag.Lookup(c).Value.String()
}

// GetWorkingDirectory returns a working directory passed through the -w option with a fallback to a process working directory.
func GetWorkingDirectory() string {
	const w = "w"

	if flag.Lookup(w) == nil {
		flag.String(w, "", "Working directory.")
		flag.Parse()
	}

	dir := flag.Lookup(w).Value.String()
	if dir != "" {
		return dir
	}

	dir, err := os.Getwd()
	if err != nil {
		return "./"
	}

	return dir
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
