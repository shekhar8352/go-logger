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
}

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
