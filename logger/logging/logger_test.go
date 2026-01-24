package logging

import (
	"context"
	"os"
	"testing"
)

// BenchmarkLogger_Info measures the performance of a standard Info log call
// We discard output to focus on the logger's internal overhead (formatting, context handling, JSON marshaling)
func BenchmarkLogger_Info(b *testing.B) {
	// Setup: Discard output to avoid disk I/O skewing results
	cfg := DefaultConfig()
	cfg.IncludeConsole = false
	// We point to /dev/null for file output simulation or just set logFile to nil if we want to avoid I/O entirely.
	// However, the logger writes to a file by default. Let's use a temp file and discard writes to it if possible,
	// or just accept I/O is part of the benchmark.
	// For pure allocation/cpu test, we might want a NopWriter.
	// But since StructuredLogger takes a config with file paths, let's just let it write to a temp file
	// and clean it up.

	// Using a temp file for realism
	tmpFile, _ := os.CreateTemp("", "bench-log-*.log")
	defer os.Remove(tmpFile.Name())

	// We override internal logFile to devNull for pure overhead test
	l := NewLogger(cfg)
	l.logFile = nil // Disable file write to test formatting/allocs primarily
	// If we want to test with file write, we should keep it.
	// Let's create a purely discard logger for this specific benchmark to check allocation of the logic.
	l.logFile = os.NewFile(0, "devnull") // Invalid but non-nil, or just mock it.
	// Actually, os.Discard is an io.Writer. The logger uses *os.File.
	// So we can open /dev/null
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	l.logFile = devNull
	defer l.Close()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info(ctx, "This is a benchmark log message")
	}
}

// BenchmarkLogger_Info_WithArgs measures performance with message formatting
func BenchmarkLogger_Info_WithArgs(b *testing.B) {
	cfg := DefaultConfig()
	cfg.IncludeConsole = false
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)

	l := NewLogger(cfg)
	l.logFile = devNull
	defer l.Close()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info(ctx, "User %s performed action %s in %d ms", "user123", "login", 42)
	}
}

// BenchmarkLogger_Silent measures performance when the log level should be skipped
func BenchmarkLogger_Silent(b *testing.B) {
	cfg := DefaultConfig()
	cfg.MinLevel = LevelError // Only Error allowed
	cfg.IncludeConsole = false

	l := NewLogger(cfg)
	// No need to set file, it shouldn't write
	defer l.Close()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// These calls should be skipped
		l.Info(ctx, "This should be ignored")
	}
}

// BenchmarkLogger_JSONMarshal measures just the overhead of creating entry and marshaling
// This isolates the logic from the lock and file I/O
func BenchmarkLogger_CreateEntry(b *testing.B) {
	cfg := DefaultConfig()
	l := &StructuredLogger{config: cfg}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.createLogEntry(ctx, LevelInfo, "Test message")
	}
}
