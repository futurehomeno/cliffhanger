package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/futurehomeno/cliffhanger/formatters"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// GetConfigurationDirectory returns a configuration directory passed through the -c option with a fallback to a relative path.
func GetConfigurationDirectory() string {
	const c = "c"

	if flag.Lookup(c) == nil {
		flag.String(c, "", "Configuration directory.")
		flag.Parse()
	}

	dir := flag.Lookup(c).Value.String()
	if dir != "" {
		return dir
	}

	return "./"
}

// GetWorkingDirectory returns a working directory with a fallback to a relative path.
func GetWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "./"
	}

	return dir
}

// InitializeLogger initializes logger with an optional log rotation.
func InitializeLogger(logFile string, level string, logFormat string) error {
	if logFile == "" {
		return fmt.Errorf("logfile not set")
	}

	switch logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	case "budzik":
		log.SetFormatter(formatters.NewBudzikFormatter())
	default:
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	l := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5, // MiB
		MaxBackups: 4,
	}

	f, err := os.OpenFile(l.Filename, os.O_RDONLY|os.O_CREATE, 0644) //nolint:gosec

	if err != nil {
		if err := os.MkdirAll(filepath.Dir(l.Filename), 0755); err != nil { //nolint:gosec
			return fmt.Errorf("create log dir=%s err: %w", filepath.Dir(l.Filename), err)
		}

		if f, err = os.OpenFile(l.Filename, os.O_RDONLY|os.O_CREATE, 0644); err != nil { //nolint:gosec
			return fmt.Errorf("open log file=%s err: %w", l.Filename, err)
		}
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("close err: %v", err)
		}
	}()
	log.SetOutput(l)

	return nil
}

// WaitForShutdown blocks code execution until a shutdown signal occurs.
func WaitForShutdown() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	<-signals
}
