package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
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

type LogEntry struct {
	Level      string
	LevelValue Level
	Message    string
}

type Subscriber chan LogEntry

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[Subscriber]struct{}
}

var global *Logger

var globalBroadcaster = &Broadcaster{
	subscribers: make(map[Subscriber]struct{}),
}

func New(levelStr string) (*Logger, error) {
	level, err := ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	flags := log.LstdFlags
	if os.Getenv("INVOCATION_ID") != "" {
		flags = 0
	}

	return &Logger{
		level:  level,
		logger: log.New(os.Stderr, "", flags),
	}, nil
}

func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %s", levelStr)
	}
}

func levelToString(level Level) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "warn"
	}
}

func SetGlobal(l *Logger) {
	global = l
}

func (b *Broadcaster) Subscribe() Subscriber {
	ch := make(Subscriber, 100)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[ch] = struct{}{}
	return ch
}

func (b *Broadcaster) Unsubscribe(ch Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, ch)
	close(ch)
}

func (b *Broadcaster) publish(entry LogEntry) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- entry:
		default:
			// drop if subscriber is slow, never block the logger
		}
	}
}

func Subscribe() Subscriber     { return globalBroadcaster.Subscribe() }
func Unsubscribe(ch Subscriber) { globalBroadcaster.Unsubscribe(ch) }

func (l *Logger) log(level Level, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)

	globalBroadcaster.publish(LogEntry{
		Level:      levelToString(level),
		LevelValue: level,
		Message:    msg,
	})

	if l.level <= level {
		l.logger.Printf("[%s] %s", strings.ToUpper(levelToString(level)), msg)
	}
}

func (l *Logger) Debug(format string, v ...any) { l.log(LevelDebug, format, v...) }
func (l *Logger) Info(format string, v ...any)  { l.log(LevelInfo, format, v...) }
func (l *Logger) Warn(format string, v ...any)  { l.log(LevelWarn, format, v...) }
func (l *Logger) Error(format string, v ...any) { l.log(LevelError, format, v...) }

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
