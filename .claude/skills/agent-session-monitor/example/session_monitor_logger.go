package main

import (
	"io"
	"log"
	"os"
	"sync"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger is a leveled logger for the session monitor.
type Logger struct {
	mu    sync.Mutex
	level LogLevel
	out   *log.Logger
}

var defaultLogger = NewLogger(LevelInfo, os.Stdout)

// NewLogger creates a Logger that writes to w at the given minimum level.
func NewLogger(level LogLevel, w io.Writer) *Logger {
	return &Logger{
		level: level,
		out:   log.New(w, "", log.LstdFlags|log.LUTC),
	}
}

// SetLevel changes the minimum log level at runtime.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level < l.level {
		return
	}
	prefix := "[" + levelNames[level] + "] "
	if len(args) == 0 {
		l.out.Print(prefix + format)
	} else {
		l.out.Printf(prefix+format, args...)
	}
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs a message at INFO level.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Package-level helpers that delegate to the default logger.

func Debug(format string, args ...interface{}) { defaultLogger.Debug(format, args...) }
func Info(format string, args ...interface{})  { defaultLogger.Info(format, args...) }
func Warn(format string, args ...interface{})  { defaultLogger.Warn(format, args...) }
func Error(format string, args ...interface{}) { defaultLogger.Error(format, args...) }

// InitLogger initialises the default logger from a Config.
func InitLogger(cfg *Config) {
	level := LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = LevelDebug
	case "warn":
		level = LevelWarn
	case "error":
		level = LevelError
	}
	defaultLogger = NewLogger(level, os.Stdout)
}
