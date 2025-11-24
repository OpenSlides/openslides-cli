package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantErr   bool
		wantLevel Level
	}{
		{"debug level", "debug", false, LevelDebug},
		{"info level", "info", false, LevelInfo},
		{"warn level", "warn", false, LevelWarn},
		{"warning level", "warning", false, LevelWarn},
		{"error level", "error", false, LevelError},
		{"invalid level", "invalid", true, 0},
		{"empty level", "", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger.level != tt.wantLevel {
				t.Errorf("New() level = %v, want %v", logger.level, tt.wantLevel)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  Level
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{"debug at debug", LevelDebug, func(l *Logger) { l.Debug("test") }, true},
		{"info at debug", LevelDebug, func(l *Logger) { l.Info("test") }, true},
		{"info at info", LevelInfo, func(l *Logger) { l.Info("test") }, true},
		{"debug at info", LevelInfo, func(l *Logger) { l.Debug("test") }, false},
		{"warn at warn", LevelWarn, func(l *Logger) { l.Warn("test") }, true},
		{"info at warn", LevelWarn, func(l *Logger) { l.Info("test") }, false},
		{"error at error", LevelError, func(l *Logger) { l.Error("test") }, true},
		{"warn at error", LevelError, func(l *Logger) { l.Warn("test") }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				level:  tt.logLevel,
				logger: log.New(&buf, "", 0),
			}

			tt.logFunc(logger)

			output := buf.String()
			if tt.shouldLog && output == "" {
				t.Error("Expected log output but got none")
			}
			if !tt.shouldLog && output != "" {
				t.Errorf("Expected no log output but got: %s", output)
			}
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  LevelDebug,
		logger: log.New(&buf, "", 0),
	}

	SetGlobal(logger)
	Debug("test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Error("Global logger did not log message")
	}
}
