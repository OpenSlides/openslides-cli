package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	level  Level
	logger *log.Logger
}

var global *Logger

func New(levelStr string) (*Logger, error) {
	var level Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = LevelDebug
	case "info":
		level = LevelInfo
	case "warn", "warning":
		level = LevelWarn
	case "error":
		level = LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", levelStr)
	}

	return &Logger{
		level:  level,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}, nil
}

func SetGlobal(l *Logger) {
	global = l
}

func (l *Logger) Debug(format string, v ...any) {
	if l.level <= LevelDebug {
		l.logger.Printf("[DEBUG] "+format, v...)
	}
}

func (l *Logger) Info(format string, v ...any) {
	if l.level <= LevelInfo {
		l.logger.Printf("[INFO] "+format, v...)
	}
}

func (l *Logger) Warn(format string, v ...any) {
	if l.level <= LevelWarn {
		l.logger.Printf("[WARN] "+format, v...)
	}
}

func (l *Logger) Error(format string, v ...any) {
	if l.level <= LevelError {
		l.logger.Printf("[ERROR] "+format, v...)
	}
}

// Global logging functions
func Debug(format string, v ...any) {
	if global != nil {
		global.Debug(format, v...)
	}
}

func Info(format string, v ...any) {
	if global != nil {
		global.Info(format, v...)
	}
}

func Warn(format string, v ...any) {
	if global != nil {
		global.Warn(format, v...)
	}
}

func Error(format string, v ...any) {
	if global != nil {
		global.Error(format, v...)
	}
}
