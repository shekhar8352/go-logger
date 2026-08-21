package logging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestLogLine(t *testing.T, dir, prefix, date, message, level string) string {
	t.Helper()
	return writeTestLogLineTS(t, dir, prefix, date, "2026-08-18T12:00:00Z", message, level)
}

func writeTestLogLineTS(t *testing.T, dir, prefix, date, timestamp, message, level string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := datedLogFile(dir, prefix, date)
	line := `{"timestamp":"` + timestamp + `","log_id":"test-id","level":"` + level + `","message":"` + message + `","file":"reader_test.go","line":1}`
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

func TestSearchLogs_WithPagination_ReturnsCorrectSlice(t *testing.T) {
	dir := t.TempDir()
	date := "2026-08-18"
	cfg := LoggerConfig{LogsDir: dir, FilePrefix: "app"}
	for _, ts := range []string{
		"2026-08-18T12:00:01Z",
		"2026-08-18T12:00:02Z",
		"2026-08-18T12:00:03Z",
		"2026-08-18T12:00:04Z",
		"2026-08-18T12:00:05Z",
	} {
		writeTestLogLineTS(t, dir, "app", date, ts, "page-item", LevelInfo)
	}

	logs, total, err := SearchLogsWithOptions(SearchOptions{
		Config: cfg,
		Query:  "page-item",
		Date:   date,
		Offset: 1,
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("SearchLogsWithOptions: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(logs) != 2 {
		t.Fatalf("page len = %d, want 2", len(logs))
	}
	// Newest-first: 05, 04, 03, 02, 01 → offset 1 limit 2 is 04 then 03.
	if logs[0].Timestamp != "2026-08-18T12:00:04Z" || logs[1].Timestamp != "2026-08-18T12:00:03Z" {
		t.Fatalf("unexpected page: %+v", logs)
	}
}

func TestReadLogs_WithTimeRange_Filters(t *testing.T) {
	dir := t.TempDir()
	date := "2026-08-18"
	writeTestLogLineTS(t, dir, "app", date, "2026-08-18T10:00:00Z", "early", LevelInfo)
	writeTestLogLineTS(t, dir, "app", date, "2026-08-18T12:00:00Z", "mid", LevelInfo)
	writeTestLogLineTS(t, dir, "app", date, "2026-08-18T14:00:00Z", "late", LevelInfo)

	start, _ := time.Parse(time.RFC3339, "2026-08-18T11:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-08-18T13:00:00Z")
	logs, total, err := ReadLogsWithOptions(ReadOptions{
		Config:    LoggerConfig{LogsDir: dir, FilePrefix: "app"},
		Date:      date,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		t.Fatalf("ReadLogsWithOptions: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Message != "mid" {
		t.Fatalf("expected only mid entry, total=%d logs=%+v", total, logs)
	}
}

func TestTail_ReturnsNewLines(t *testing.T) {
	dir := t.TempDir()
	cfg := LoggerConfig{
		LogsDir:        dir,
		FilePrefix:     "app",
		IncludeConsole: false,
		MinLevel:       LevelInfo,
	}
	lg := NewLogger(cfg)
	defer lg.Close()
	lg.Info(context.Background(), "already-there")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := TailLogs(ctx, TailOptions{Config: cfg, PollEvery: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("TailLogs: %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	lg.Info(context.Background(), "brand-new-line")

	deadline := time.After(2 * time.Second)
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				t.Fatal("tail channel closed before new line")
			}
			if strings.Contains(e.Message, "already-there") {
				t.Fatal("tail should start at EOF, not replay existing lines")
			}
			if strings.Contains(e.Message, "brand-new-line") {
				return
			}
		case <-deadline:
			t.Fatal("did not receive tailed line")
		}
	}
}

func TestSearchLogs_WithRegex_Matches(t *testing.T) {
	dir := t.TempDir()
	date := time.Now().Format("2006-01-02")
	writeTestLogLine(t, dir, "app", date, "user-42 logged in", LevelInfo)
	writeTestLogLine(t, dir, "app", date, "user-abc logged in", LevelInfo)
	writeTestLogLine(t, dir, "app", date, "unrelated", LevelInfo)

	cfg := LoggerConfig{LogsDir: dir, FilePrefix: "app"}
	logs, total, err := SearchLogsWithOptions(SearchOptions{
		Config:   cfg,
		Query:    `user-\d+`,
		Date:     date,
		UseRegex: true,
	})
	if err != nil {
		t.Fatalf("regex search: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Message != "user-42 logged in" {
		t.Fatalf("expected only user-42, total=%d logs=%+v", total, logs)
	}

	plain, plainTotal, err := SearchLogsWithOptions(SearchOptions{
		Config:   cfg,
		Query:    `user-\d+`,
		Date:     date,
		UseRegex: false,
	})
	if err != nil {
		t.Fatalf("substring search: %v", err)
	}
	if plainTotal != 0 || len(plain) != 0 {
		t.Fatalf("substring search should not treat query as regex, got %+v", plain)
	}
}
