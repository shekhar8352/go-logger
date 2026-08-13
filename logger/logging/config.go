package logging

import "io"

// Log level constants
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// LoggerConfig holds configuration for the StructuredLogger
type LoggerConfig struct {
	LogsDir        string
	FilePrefix     string
	IncludeConsole bool
	MinLevel       string
	// Writer is an optional sink for log output. When set, file rotation is
	// skipped and entries are written to Writer instead. Useful for tests
	// (e.g. BufferSink) or custom destinations.
	Writer io.Writer
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		LogsDir:        "logs",
		FilePrefix:     "app",
		IncludeConsole: true,
		MinLevel:       LevelInfo,
	}
}
