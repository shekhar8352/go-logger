package logging

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotateLogFile_WhenDirFails_DoesNotPanic(t *testing.T) {
	// Use a regular file as LogsDir so MkdirAll cannot create a directory.
	blocker, err := os.CreateTemp("", "not-a-dir-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	blockerPath := blocker.Name()
	_ = blocker.Close()
	defer os.Remove(blockerPath)

	cfg := LoggerConfig{
		LogsDir:        blockerPath,
		FilePrefix:     "app",
		IncludeConsole: false,
		MinLevel:       LevelInfo,
	}

	var l *StructuredLogger
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewLogger/rotateLogFile panicked: %v", r)
			}
		}()
		l = NewLogger(cfg)
	}()

	if l == nil {
		t.Fatal("expected logger instance even when directory creation fails")
	}
	if l.RotationError() == nil {
		t.Fatal("expected RotationError when log directory cannot be created")
	}

	assertLogsWithoutPanic(t, l, "still usable after dir failure")
}

func TestRotateLogFile_WhenFileOpenFails_DoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Format("2006-01-02")
	// Create a directory with the log file name so OpenFile fails.
	blockingPath := filepath.Join(dir, "app-"+today+".log")
	if err := os.Mkdir(blockingPath, 0o755); err != nil {
		t.Fatalf("mkdir blocking path: %v", err)
	}

	cfg := LoggerConfig{
		LogsDir:        dir,
		FilePrefix:     "app",
		IncludeConsole: false,
		MinLevel:       LevelInfo,
	}

	var l *StructuredLogger
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewLogger/rotateLogFile panicked: %v", r)
			}
		}()
		l = NewLogger(cfg)
	}()

	if l == nil {
		t.Fatal("expected logger instance even when file open fails")
	}
	if l.RotationError() == nil {
		t.Fatal("expected RotationError when log file cannot be opened")
	}

	assertLogsWithoutPanic(t, l, "still usable after file open failure")
}

func assertLogsWithoutPanic(t *testing.T, l *StructuredLogger, msg string) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("logging after rotation failure panicked: %v", rec)
			}
		}()
		l.Info(context.Background(), msg)
	}()

	_ = w.Close()
	out, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !bytes.Contains(out, []byte(msg)) {
		t.Fatalf("expected fallback stderr write to contain %q, got %q", msg, out)
	}
}
