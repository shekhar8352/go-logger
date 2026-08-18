package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestLogLine(t *testing.T, dir, prefix, date, message, level string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := datedLogFile(dir, prefix, date)
	line := `{"timestamp":"2026-08-18T12:00:00Z","log_id":"test-id","level":"` + level + `","message":"` + message + `","file":"reader_test.go","line":1}`
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		_ = f.Close()
		t.Fatalf("write log file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close log file: %v", err)
	}
	return path
}

func TestSearchLogs_WithConfig_UsesConfigDirAndPrefix(t *testing.T) {
	dir := t.TempDir()
	date := time.Now().Format("2006-01-02")
	prefix := "svc"
	token := "unique-search-token-phase1"

	writeTestLogLine(t, dir, prefix, date, token, LevelInfo)

	cfg := LoggerConfig{LogsDir: dir, FilePrefix: prefix}
	logs, count, err := SearchLogsWithConfig(cfg, token, date, "")
	if err != nil {
		t.Fatalf("SearchLogsWithConfig: %v", err)
	}
	if count != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 match in configured path, got count=%d logs=%d", count, len(logs))
	}
	if logs[0].Message != token {
		t.Fatalf("unexpected message: %q", logs[0].Message)
	}

	// Empty date still looks at the last 7 days under the configured prefix/dir.
	logs, count, err = SearchLogsWithConfig(cfg, token, "", "")
	if err != nil {
		t.Fatalf("SearchLogsWithConfig (no date): %v", err)
	}
	if count != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 match without date, got count=%d logs=%d", count, len(logs))
	}

	// Default SearchLogs must not pick up the custom-dir/prefix file.
	logs, count, err = SearchLogs(token, date, "")
	if err != nil {
		t.Fatalf("SearchLogs default: %v", err)
	}
	if count != 0 || len(logs) != 0 {
		t.Fatalf("default SearchLogs should ignore custom dir/prefix, got count=%d", count)
	}
}

func TestReadLogsByDate_WithConfig_UsesConfigPath(t *testing.T) {
	dir := t.TempDir()
	date := "1999-12-31"
	prefix := "orders"
	msg := "read-by-date-config-message"

	writeTestLogLine(t, dir, prefix, date, msg, LevelError)

	cfg := LoggerConfig{LogsDir: dir, FilePrefix: prefix}
	logs, count, err := ReadLogsByDateWithConfig(cfg, date, "")
	if err != nil {
		t.Fatalf("ReadLogsByDateWithConfig: %v", err)
	}
	if count != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 log from config path, got count=%d logs=%d", count, len(logs))
	}
	if logs[0].Message != msg {
		t.Fatalf("unexpected message: %q", logs[0].Message)
	}

	// Level filter still applies with config.
	logs, count, err = ReadLogsByDateWithConfig(cfg, date, LevelInfo)
	if err != nil {
		t.Fatalf("ReadLogsByDateWithConfig level filter: %v", err)
	}
	if count != 0 || len(logs) != 0 {
		t.Fatalf("expected no INFO logs, got count=%d", count)
	}

	// Default reader still looks at GetLogDirectory()/app-<date>.log, not the temp path.
	_, _, err = ReadLogsByDate(date, "")
	if err == nil {
		t.Fatal("expected default ReadLogsByDate to miss the custom path")
	}
	if !strings.Contains(err.Error(), "log file not found") {
		t.Fatalf("expected not-found error for default path, got %v", err)
	}
}

func TestGetLogDirectory_RespectsEnv(t *testing.T) {
	want := filepath.Join(string(os.PathSeparator), "var", "log", "myapp")
	t.Setenv("LOG_DIR", want)
	if got := GetLogDirectory(); got != want {
		t.Fatalf("GetLogDirectory() = %q, want %q", got, want)
	}

	t.Setenv("LOG_DIR", "")
	if got := GetLogDirectory(); got != "./logs" {
		t.Fatalf("GetLogDirectory() with empty LOG_DIR = %q, want %q", got, "./logs")
	}
}

func TestGetAvailableLogFiles_WithCustomPrefix(t *testing.T) {
	dir := t.TempDir()
	prefix := "custom"

	writeTestLogLine(t, dir, prefix, "2026-08-18", "newer", LevelInfo)
	writeTestLogLine(t, dir, prefix, "2026-08-17", "older", LevelInfo)
	writeTestLogLine(t, dir, "app", "2026-08-18", "default-prefix", LevelInfo)
	if err := os.WriteFile(filepath.Join(dir, "custom-not-a-date.log"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write decoy: %v", err)
	}

	files, err := GetAvailableLogFilesWithConfig(LoggerConfig{LogsDir: dir, FilePrefix: prefix})
	if err != nil {
		t.Fatalf("GetAvailableLogFilesWithConfig: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 custom-prefix files, got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "custom-2026-08-18.log" {
		t.Fatalf("expected newest file first, got %q", files[0])
	}
	if filepath.Base(files[1]) != "custom-2026-08-17.log" {
		t.Fatalf("expected older file second, got %q", files[1])
	}

	appFiles, err := GetAvailableLogFilesWithConfig(LoggerConfig{LogsDir: dir, FilePrefix: "app"})
	if err != nil {
		t.Fatalf("GetAvailableLogFilesWithConfig app: %v", err)
	}
	if len(appFiles) != 1 || filepath.Base(appFiles[0]) != "app-2026-08-18.log" {
		t.Fatalf("expected only app-prefixed file, got %v", appFiles)
	}
}
