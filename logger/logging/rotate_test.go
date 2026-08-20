package logging

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

func fileLoggerConfig(dir string) LoggerConfig {
	return LoggerConfig{
		LogsDir:        dir,
		FilePrefix:     "app",
		IncludeConsole: false,
		MinLevel:       LevelInfo,
	}
}

func listLogNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestRotate_WhenSizeExceeded_RotatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := fileLoggerConfig(dir)
	cfg.MaxFileSize = 1

	lg := NewLogger(cfg)
	defer lg.Close()
	if lg.RotationError() != nil {
		t.Fatalf("RotationError: %v", lg.RotationError())
	}

	ctx := context.Background()
	lg.Info(ctx, "first-line")
	lg.Info(ctx, "second-line")

	names := listLogNames(t, dir)
	today := time.Now().Format("2006-01-02")
	active := "app-" + today + ".log"
	archived := "app-" + today + ".1.log"

	foundActive, foundArchived := false, false
	for _, n := range names {
		if n == active {
			foundActive = true
		}
		if n == archived {
			foundArchived = true
		}
	}
	if !foundActive {
		t.Fatalf("expected active file %s, got %v", active, names)
	}
	if !foundArchived {
		t.Fatalf("expected archived file %s after size rotation, got %v", archived, names)
	}

	archivedBytes, err := os.ReadFile(filepath.Join(dir, archived))
	if err != nil {
		t.Fatalf("read archived: %v", err)
	}
	if !bytes.Contains(archivedBytes, []byte("first-line")) {
		t.Fatalf("archived file should contain first line, got %s", archivedBytes)
	}
	activeBytes, err := os.ReadFile(filepath.Join(dir, active))
	if err != nil {
		t.Fatalf("read active: %v", err)
	}
	if !bytes.Contains(activeBytes, []byte("second-line")) {
		t.Fatalf("active file should contain second line, got %s", activeBytes)
	}
}

func TestRetention_DeletesFilesOlderThanN(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "app-2000-01-01.log")
	if err := os.WriteFile(oldPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old log: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	recentPath := filepath.Join(dir, "app-"+today+".log")
	if err := os.WriteFile(recentPath, []byte("recent\n"), 0o644); err != nil {
		t.Fatalf("write recent log: %v", err)
	}

	cfg := fileLoggerConfig(dir)
	cfg.RetentionDays = 1
	lg := NewLogger(cfg)
	defer lg.Close()
	if lg.RotationError() != nil {
		t.Fatalf("RotationError: %v", lg.RotationError())
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old log to be deleted, stat err=%v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Fatalf("expected recent log to be kept: %v", err)
	}
}

func TestCompressRotated_CreatesGzipFile(t *testing.T) {
	dir := t.TempDir()
	cfg := fileLoggerConfig(dir)
	cfg.MaxFileSize = 1
	cfg.CompressRotated = true

	lg := NewLogger(cfg)
	defer lg.Close()

	ctx := context.Background()
	lg.Info(ctx, "compress-me")
	lg.Info(ctx, "after-rotate")

	today := time.Now().Format("2006-01-02")
	gzPath := filepath.Join(dir, "app-"+today+".1.log.gz")
	plainArchived := filepath.Join(dir, "app-"+today+".1.log")
	active := filepath.Join(dir, "app-"+today+".log")

	if _, err := os.Stat(gzPath); err != nil {
		t.Fatalf("expected gzip archive %s, files=%v err=%v", gzPath, listLogNames(t, dir), err)
	}
	if _, err := os.Stat(plainArchived); !os.IsNotExist(err) {
		t.Fatal("uncompressed archived log should be removed after gzip")
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatalf("active log should remain uncompressed: %v", err)
	}

	f, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("open gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	payload, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if !bytes.Contains(payload, []byte("compress-me")) {
		t.Fatalf("gzip payload missing original line: %s", payload)
	}
}

func TestRotation_DoesNotLoseLines(t *testing.T) {
	dir := t.TempDir()
	cfg := fileLoggerConfig(dir)
	cfg.MaxFileSize = 1

	lg := NewLogger(cfg)
	defer lg.Close()

	const n = 12
	ctx := context.Background()
	for i := 0; i < n; i++ {
		lg.Info(ctx, "keep-line-%d", i)
	}

	var all []byte
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, readErr := os.ReadFile(filepath.Join(dir, e.Name()))
		if readErr != nil {
			t.Fatalf("read %s: %v", e.Name(), readErr)
		}
		all = append(all, b...)
	}

	for i := 0; i < n; i++ {
		want := "keep-line-" + strconv.Itoa(i)
		if !bytes.Contains(all, []byte(want)) {
			t.Fatalf("missing %q in rotated files %v\n%s", want, listLogNames(t, dir), all)
		}
	}
}
