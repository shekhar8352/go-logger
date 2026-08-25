package logging

import (
	"io"
	"time"
)

// Log level constants
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// Log format constants
const (
	FormatJSON   = "json"
	FormatLogfmt = "logfmt"
)

// TimestampUnixMs formats timestamps as Unix milliseconds (decimal string).
// Any other TimestampFormat value is treated as a time.Format layout;
// empty defaults to time.RFC3339.
const TimestampUnixMs = "unix_ms"

// LoggerConfig holds configuration for the StructuredLogger
type LoggerConfig struct {
	LogsDir        string
	FilePrefix     string
	IncludeConsole bool
	MinLevel       string
	// LogFormat selects the line format: FormatJSON (default) or FormatLogfmt.
	LogFormat string
	// TimestampFormat is a time.Format layout, or TimestampUnixMs.
	// Empty defaults to time.RFC3339.
	TimestampFormat string
	// Writer is an optional sink for log output. When set, file rotation is
	// skipped and entries are written to Writer instead. Useful for tests
	// (e.g. BufferSink) or custom destinations.
	Writer io.Writer
	// MaxFileSize is the size in bytes at which the current log file is
	// rotated. Zero disables size-based rotation (daily rotation still applies).
	MaxFileSize int64
	// RetentionDays deletes log files whose encoded date is older than this
	// many days. Zero disables age-based cleanup.
	RetentionDays int
	// MaxBackups is the maximum number of rotated/archived files to keep
	// (the active file is not counted). Zero disables count-based cleanup.
	MaxBackups int
	// CompressRotated gzip-compresses files that have been rotated away
	// (e.g. app-2026-01-22.log.gz). The active log file is never compressed.
	CompressRotated bool
	// UseAsyncWriter buffers log lines and writes them in a background
	// goroutine. Close() flushes remaining buffered lines.
	UseAsyncWriter bool
	// AsyncBufferSize is the async channel capacity. Zero defaults to 256.
	AsyncBufferSize int
	// FlushInterval, when greater than zero, Sync/Flushes outputs on this
	// cadence in addition to writes. Zero writes as soon as the worker runs.
	FlushInterval time.Duration
	// StderrLevels lists levels written to stderr when IncludeConsole is true.
	// Other console levels go to stdout. Empty means all console output to stdout.
	StderrLevels []string
	// Sinks receive a copy of every written log line, in addition to the
	// primary file or Writer and optional console output.
	Sinks []io.Writer
	// Hooks are invoked asynchronously for each entry that passes MinLevel.
	// They must not panic; panics are recovered. They do not block logging.
	Hooks []Hook
	// OnError is an optional hook invoked only for ERROR entries, with the
	// same async/non-blocking contract as Hooks.
	OnError Hook
	// SampleEveryN, when greater than 1, keeps 1 in N entries for the levels
	// in SampleLevels (deterministic: the 1st, N+1st, …). Zero or 1 disables
	// sampling so every entry that passes MinLevel is logged.
	SampleEveryN int
	// SampleLevels are the levels sampling applies to. Empty means DEBUG only.
	SampleLevels []string
	// IncludeStackTraceOnError attaches a stack trace to ERROR and WARN
	// entries when true. Other levels are unchanged. False (the default)
	// omits the stack_trace field from output.
	IncludeStackTraceOnError bool
}

// Hook is called with a log entry after MinLevel filtering.
// Hooks run in the background so they must not assume they complete before
// the next log call, and they should avoid long-running work when possible.
type Hook func(LogEntry)

// DefaultConfig returns a sensible default configuration
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		LogsDir:         "logs",
		FilePrefix:      "app",
		IncludeConsole:  true,
		MinLevel:        LevelInfo,
		LogFormat:       FormatJSON,
		TimestampFormat: time.RFC3339,
	}
}
