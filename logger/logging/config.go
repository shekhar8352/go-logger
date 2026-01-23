package logging

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
