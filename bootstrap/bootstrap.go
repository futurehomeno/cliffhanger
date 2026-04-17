package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/futurehomeno/cliffhanger/formatters"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logOutputLock    sync.Mutex
	currentLogOutput *lumberjack.Logger
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
	SetLogFormat(logFormat)

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	return SetLogOutput(logFile)
}

// SetLogFormat sets the logrus formatter matching the given format name.
// Unknown formats fall back to the default text formatter.
func SetLogFormat(logFormat string) {
	switch logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	case "budzik":
		log.SetFormatter(formatters.NewBudzikFormatter())
	default:
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}
}

// SetLogOutput (re)configures the lumberjack-rotated log output to the given
// file. It creates the target directory if missing and closes any previously
// configured output. Safe to call repeatedly at runtime.
func SetLogOutput(logFile string) error {
	if logFile == "" {
		return fmt.Errorf("logfile not set")
	}

	logOutputLock.Lock()
	defer logOutputLock.Unlock()

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil { //nolint:gosec
		return fmt.Errorf("create log dir=%s err: %w", filepath.Dir(logFile), err)
	}

	if f, err := os.OpenFile(logFile, os.O_RDONLY|os.O_CREATE, 0644); err != nil { //nolint:gosec
		return fmt.Errorf("open log file=%s err: %w", logFile, err)
	} else if cerr := f.Close(); cerr != nil {
		log.Errorf("close err: %v", cerr)
	}

	newOutput := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5, // MiB
		MaxBackups: 4,
	}

	previous := currentLogOutput
	currentLogOutput = newOutput
	log.SetOutput(newOutput)

	if previous != nil && previous != newOutput {
		if err := previous.Close(); err != nil {
			log.Errorf("close previous log output err: %v", err)
		}
	}

	return nil
}

// WaitForShutdown blocks code execution until a shutdown signal occurs.
func WaitForShutdown() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	<-signals
}
