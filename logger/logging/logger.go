package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// StructuredLogger handles structured logging with rotation and context support.
// Child loggers created with WithField/WithFields share the parent's writer and
// add their fields to every log entry.
type StructuredLogger struct {
	mu          sync.RWMutex
	logFile     *os.File
	currentDay  string
	currentSize int64
	config      LoggerConfig
	rotateErr   error
	fields      map[string]interface{}
	root        *StructuredLogger

	asyncCh   chan logJob
	asyncDone chan struct{}
	sendMu    sync.Mutex
	closed    bool

	debugCount atomic.Uint64
	infoCount  atomic.Uint64
	warnCount  atomic.Uint64
	errorCount atomic.Uint64
}

type logJob struct {
	line  []byte
	data  []byte
	level string
}

// NewLogger creates a new StructuredLogger using the provided config.
// When cfg.Writer is set, file rotation is skipped. Otherwise a log file is
// opened; if that fails the logger falls back to stderr and remains usable.
func NewLogger(cfg LoggerConfig) *StructuredLogger {
	l := &StructuredLogger{config: cfg}
	if cfg.Writer == nil {
		_ = l.rotateLogFile()
	}
	if cfg.UseAsyncWriter {
		size := cfg.AsyncBufferSize
		if size <= 0 {
			size = 256
		}
		l.asyncCh = make(chan logJob, size)
		l.asyncDone = make(chan struct{})
		go l.asyncLoop()
	}
	return l
}

// RotationError returns the last error from creating the log directory or
// opening the log file. It is nil when rotation has succeeded or when a
// custom Writer is in use.
func (l *StructuredLogger) RotationError() error {
	w := l.rootOrSelf()
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rotateErr
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

// WithField returns a child of the default logger that includes key on every entry.
func WithField(key string, value interface{}) *StructuredLogger {
	return DefaultLogger.WithField(key, value)
}

// WithFields returns a child of the default logger that includes fields on every entry.
func WithFields(fields map[string]interface{}) *StructuredLogger {
	return DefaultLogger.WithFields(fields)
}

func (l *StructuredLogger) rootOrSelf() *StructuredLogger {
	if l.root != nil {
		return l.root
	}
	return l
}

// WithField returns a child logger that adds key to every log entry.
// The child shares the parent's output and configuration.
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	return l.WithFields(map[string]interface{}{key: value})
}

// WithFields returns a child logger that adds fields to every log entry.
// Child fields overlay the parent's fields. Incoming keys overwrite existing ones.
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	return &StructuredLogger{
		root:   l.rootOrSelf(),
		fields: mergeFields(copyFields(l.fields), copyFields(fields)),
	}
}

// SetLevel changes the minimum log level for the logger
func (l *StructuredLogger) SetLevel(level string) {
	w := l.rootOrSelf()
	w.mu.Lock()
	defer w.mu.Unlock()
	w.config.MinLevel = level
}

// Close cleans up file handles. Closing a child logger is a no-op; close the parent.
func (l *StructuredLogger) Close() error {
	if l.root != nil {
		return nil
	}

	l.sendMu.Lock()
	if l.closed {
		l.sendMu.Unlock()
		return nil
	}
	l.closed = true
	ch := l.asyncCh
	done := l.asyncDone
	l.sendMu.Unlock()

	if ch != nil {
		close(ch)
		<-done
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil
		return err
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
	w := l.rootOrSelf()
	return levelPriority(level) >= levelPriority(w.config.MinLevel)
}

// writeLog writes a LogEntry to file and console (if configured)
func (l *StructuredLogger) writeLog(entry LogEntry) {
	if !l.shouldLog(entry.Level) {
		return
	}

	w := l.rootOrSelf()
	w.recordMetrics(entry.Level)
	w.invokeHooks(entry)

	data := encodeLogLine(w.config.LogFormat, entry)
	line := make([]byte, len(data)+1)
	copy(line, data)
	line[len(data)] = '\n'
	job := logJob{line: line, data: data, level: entry.Level}

	w.sendMu.Lock()
	if w.closed {
		w.sendMu.Unlock()
		return
	}
	if w.asyncCh != nil {
		w.asyncCh <- job
		w.sendMu.Unlock()
		return
	}
	w.sendMu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	w.emitLocked(job)
}

func (l *StructuredLogger) emitLocked(job logJob) {
	if l.config.Writer != nil {
		_, _ = l.config.Writer.Write(job.line)
	} else {
		_ = l.rotateLogFile()
		if l.logFile != nil {
			n, _ := l.logFile.Write(job.line)
			l.currentSize += int64(n)
		} else {
			_, _ = os.Stderr.Write(job.line)
		}
	}

	for _, s := range l.config.Sinks {
		if s != nil {
			_, _ = s.Write(job.line)
		}
	}

	if l.config.IncludeConsole {
		writeConsole(job.level, job.data, l.config.StderrLevels)
	}
}

func writeConsole(level string, data []byte, stderrLevels []string) {
	out := os.Stdout
	if consoleGoesToStderr(level, stderrLevels) {
		out = os.Stderr
	}
	fmt.Fprintln(out, string(data))
}

func consoleGoesToStderr(level string, stderrLevels []string) bool {
	for _, l := range stderrLevels {
		if strings.EqualFold(l, level) {
			return true
		}
	}
	return false
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
		Timestamp: formatTimestamp(time.Now(), l.rootOrSelf().config.TimestampFormat),
		LogID:     GetLogID(ctx),
		RequestID: GetRequestID(ctx),
		TraceID:   GetTraceID(ctx),
		UserID:    GetUserID(ctx),
		Level:     level,
		Message:   msg,
		File:      file,
		Line:      line,
		Function:  function,
		Fields:    mergeFields(copyFields(l.fields), GetContextFields(ctx)),
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
