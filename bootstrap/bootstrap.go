package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ServiceLogField is a log field name for a service name.
const ServiceLogField = "service"

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
func InitializeLogger(logFile string, level string, logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	log.AddHook(loggerServiceHook{})

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

type loggerServiceHook struct{}

// Levels returns a list of all log levels.
func (m loggerServiceHook) Levels() []log.Level {
	return []log.Level{
		log.TraceLevel,
		log.DebugLevel,
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

// Fire adds a service name to a log entry.
func (m loggerServiceHook) Fire(entry *log.Entry) error {
	service, ok := entry.Data[ServiceLogField]
	if !ok {
		return nil
	}

	delete(entry.Data, ServiceLogField)

	entry.Message = fmt.Sprintf("[%s] %s", service, entry.Message)

	return nil
}
