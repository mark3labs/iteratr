package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents a log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a log level string
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", s)
	}
}

// Logger is a simple leveled logger
type Logger struct {
	mu     sync.Mutex
	level  Level
	logger *log.Logger
	file   *os.File
}

var (
	// Default is the default logger instance
	Default *Logger
)

func init() {
	Default = New()
}

// New creates a new logger based on environment variables
func New() *Logger {
	l := &Logger{
		level:  LevelInfo,
		logger: log.New(io.Discard, "", log.LstdFlags),
	}

	// Read log level from environment
	if levelStr := os.Getenv("ITERATR_LOG_LEVEL"); levelStr != "" {
		if level, err := ParseLevel(levelStr); err == nil {
			l.level = level
		}
	}

	// Read log file from environment
	if logFile := os.Getenv("ITERATR_LOG_FILE"); logFile != "" {
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			l.file = f
			l.logger = log.New(f, "", log.LstdFlags)
		}
	}

	return l
}

// Close closes the logger and any open file handles
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LevelDebug, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LevelInfo, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(LevelWarn, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LevelError, format, v...)
}

func (l *Logger) log(level Level, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, v...)
	l.logger.Printf("[%s] %s", level, msg)
}

// Package-level functions that use the default logger

// Debug logs a debug message using the default logger
func Debug(format string, v ...interface{}) {
	Default.Debug(format, v...)
}

// Info logs an info message using the default logger
func Info(format string, v ...interface{}) {
	Default.Info(format, v...)
}

// Warn logs a warning message using the default logger
func Warn(format string, v ...interface{}) {
	Default.Warn(format, v...)
}

// Error logs an error message using the default logger
func Error(format string, v ...interface{}) {
	Default.Error(format, v...)
}

// Close closes the default logger
func Close() error {
	return Default.Close()
}
