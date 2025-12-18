package formatters

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	stdLogger        = logrus.New() // EventLogger logs boot events
	lumberjackLogger = lumberjack.Logger{}
	defaultLevel     = "info"
)

type BudzikFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}

func (f *BudzikFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return fmt.Appendf(nil, "%s %s %s\n", timestamp, f.LevelDesc[entry.Level], entry.Message), nil
}

func SetupBudzikLogger(logFile string, level string, maxSize int) {
	lvlDesc := []string{"PANIC", "FATAL", "E", "W", "I", "D", "T", "?"}

	stdLogger.SetFormatter(&BudzikFormatter{TimestampFormat: "01-02 15:04:05", LevelDesc: lvlDesc})

	logLevel, err := logrus.ParseLevel(level)
	if err == nil {
		defaultLevel = level
		stdLogger.SetLevel(logLevel)
	} else {
		stdLogger.SetLevel(logrus.DebugLevel)
	}

	if err := setupStdLogger(logFile, 30, maxSize); err != nil {
		fmt.Printf("SetupStdLogger err: %s\n", err.Error())
	}
}

func Level() string {
	return stdLogger.Level.String()
}

func SetDefaultLevel() error {
	return SetLevel(defaultLevel)
}

func SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err == nil {
		stdLogger.SetLevel(logLevel)
		return nil
	}

	return fmt.Errorf("cant parse log level err: %s", err.Error())
}

func SetMaxSize(maxSizeStr string) error {
	maxSize, err := strconv.Atoi(maxSizeStr)

	if err != nil {
		return fmt.Errorf("parse maxSizeStr err: %s", err.Error())
	}

	lumberjackLogger.MaxSize = maxSize

	f, err := os.OpenFile(lumberjackLogger.Filename, os.O_RDONLY|os.O_CREATE, 0644)

	if err != nil {
		return fmt.Errorf("couldn't open log file=%s", lumberjackLogger.Filename)
	}

	f.Close()

	stdLogger.SetOutput(&lumberjackLogger)
	return nil
}

func MaxSize() string {
	return fmt.Sprintf("%d", lumberjackLogger.MaxSize)
}

func setupStdLogger(logFile string, MaxBackups int, maxSize int) error {
	if logFile == "" {
		return fmt.Errorf("invalid log file=%s", logFile)
	}

	lumberjackLogger = lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize, // megabytes
		MaxBackups: MaxBackups,
		Compress:   true,
		LocalTime:  true,
	}

	f, err := os.OpenFile(lumberjackLogger.Filename, os.O_RDONLY|os.O_CREATE, 0644)

	if err != nil {
		if err := os.MkdirAll(filepath.Dir(lumberjackLogger.Filename), 0770); err != nil {
			return fmt.Errorf("couldn't create log dir=%s err: %s", filepath.Dir(lumberjackLogger.Filename), err.Error())
		}

		if f, err = os.OpenFile(lumberjackLogger.Filename, os.O_RDONLY|os.O_CREATE, 0644); err != nil {
			return fmt.Errorf("couldn't open log file=%s", lumberjackLogger.Filename)
		}
	}

	f.Close()

	stdLogger.SetOutput(&lumberjackLogger)
	return nil
}

func Writer() *io.PipeWriter {
	return stdLogger.Writer()
}

func Trace(args ...any) {
	stdLogger.Trace(args...)
}

func Info(args ...any) {
	stdLogger.Info(args...)
}

func Debug(args ...any) {
	stdLogger.Debug(args...)
}

func Warn(args ...any) {
	stdLogger.Warn(args...)
}

func Error(args ...any) {
	stdLogger.Error(args...)
}

func WithError(e error, msg string) {
	msgStr := msg

	if e != nil {
		msgStr += fmt.Sprintf(" err: '%s'", strings.Trim(e.Error(), "\n"))
	}

	stdLogger.Error(msgStr)
}

func Fatal(args ...any) {
	stdLogger.Fatal(args...)
}

func Fatalf(format string, args ...any) {
	stdLogger.Fatalf(format, args...)
}

func Tracef(format string, args ...any) {
	stdLogger.Tracef(format, args...)
}

func Infof(format string, args ...any) {
	stdLogger.Infof(format, args...)
}

func Debugf(format string, args ...any) {
	stdLogger.Debugf(format, args...)
}

func Warnf(format string, args ...any) {
	stdLogger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	stdLogger.Errorf(format, args...)
}

func Panicf(format string, args ...any) {
	stdLogger.Panicf(format, args...)
}

func InvalidIface(exp, act any) {
	_, fileName, fileLine, _ := runtime.Caller(1)
	stdLogger.Errorf("Invalid interface=%T exp=%T in %s:%d", exp, act, fileName, fileLine)
}

func WarnWithErrorf(e error, format string, args ...any) {
	msgStr := ""
	if args != nil {
		msgStr = fmt.Sprintf(format, args...)
	} else {
		msgStr = format
	}

	if e != nil {
		msgStr = fmt.Sprintf("%s err: '%s'", msgStr, e.Error())
	}

	stdLogger.Warn(msgStr)
}

func WithErrorf(e error, format string, args ...any) {
	msgStr := ""
	if args != nil {
		msgStr = fmt.Sprintf(format, args...)
	} else {
		msgStr = format
	}

	if e != nil {
		msgStr = fmt.Sprintf("%s err: '%s'", msgStr, strings.Trim(e.Error(), "\n"))
	}

	stdLogger.Error(msgStr)
}
