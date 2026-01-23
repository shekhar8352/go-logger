package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// StructuredLogger handles structured logging with rotation and context support
type StructuredLogger struct {
	mu         sync.RWMutex
	logFile    *os.File
	currentDay string
	config     LoggerConfig
}

// NewLogger creates a new StructuredLogger using the provided config
func NewLogger(cfg LoggerConfig) *StructuredLogger {
	l := &StructuredLogger{config: cfg}
	l.rotateLogFile()
	return l
}

// Global default instance
var DefaultLogger = NewLogger(DefaultConfig())

// Helper API to use default logger easily
func Debug(ctx context.Context, msg string, args ...interface{}) {
	DefaultLogger.Debug(ctx, msg, args...)
}
func Info(ctx context.Context, msg string, args ...interface{}) {
	DefaultLogger.Info(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...interface{}) {
	DefaultLogger.Warn(ctx, msg, args...)
}
func Error(ctx context.Context, msg string, args ...interface{}) {
	DefaultLogger.Error(ctx, msg, args...)
}

// SetLevel changes the minimum log level for the logger
func (l *StructuredLogger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.MinLevel = level
}

// Close cleans up file handles
func (l *StructuredLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// helper to map level strings to priority
func levelPriority(level string) int {
	switch strings.ToUpper(level) {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

func (l *StructuredLogger) shouldLog(level string) bool {
	return levelPriority(level) >= levelPriority(l.config.MinLevel)
}

// writeLog writes a LogEntry to file and console (if configured)
func (l *StructuredLogger) writeLog(entry LogEntry) {
	if !l.shouldLog(entry.Level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// rotate if day changed
	l.rotateLogFile()

	data, err := json.Marshal(entry)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"json marshal error: %v"}`,
			entry.Timestamp, entry.Level, err))
	}

	if l.logFile != nil {
		_, _ = l.logFile.Write(append(data, '\n'))
	}

	if l.config.IncludeConsole {
		fmt.Println(string(data))
	}
}

// getCallerInfo returns filename, line, function for a given depth
func getCallerInfo(depth int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown", 0, "unknown"
	}
	filename := filepath.Base(file)
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}
		if lastDot := strings.LastIndex(funcName, "."); lastDot >= 0 {
			funcName = funcName[lastDot+1:]
		}
	}
	return filename, line, funcName
}

// createLogEntry builds a LogEntry with caller info
func (l *StructuredLogger) createLogEntry(ctx context.Context, level string, msg string, args ...interface{}) LogEntry {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	file, line, function := getCallerInfo(3)
	return LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		LogID:     GetLogID(ctx),
		Level:     level,
		Message:   msg,
		File:      file,
		Line:      line,
		Function:  function,
	}
}

// Public logging methods on StructuredLogger
func (l *StructuredLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	entry := l.createLogEntry(ctx, LevelDebug, msg, args...)
	l.writeLog(entry)
}

func (l *StructuredLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	entry := l.createLogEntry(ctx, LevelInfo, msg, args...)
	l.writeLog(entry)
}

func (l *StructuredLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	entry := l.createLogEntry(ctx, LevelWarn, msg, args...)
	l.writeLog(entry)
}

func (l *StructuredLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	entry := l.createLogEntry(ctx, LevelError, msg, args...)
	l.writeLog(entry)
}
