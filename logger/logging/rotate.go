package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// rotateLogFile handles creating/rotating a log file for the given logger.
// Rotation happens on day change and, when MaxFileSize > 0, when the current
// file has reached that size. On failure it leaves logFile nil so subsequent
// writes fall back to stderr and later calls can retry. It never panics.
func (l *StructuredLogger) rotateLogFile() error {
	today := time.Now().Format("2006-01-02")
	dayChanged := l.logFile != nil && today != l.currentDay
	sizeExceeded := l.config.MaxFileSize > 0 && l.logFile != nil && l.currentSize >= l.config.MaxFileSize

	if l.logFile != nil && !dayChanged && !sizeExceeded {
		return nil
	}

	if l.logFile != nil {
		closedPath := l.logFile.Name()
		closedDay := l.currentDay
		_ = l.logFile.Close()
		l.logFile = nil
		l.currentSize = 0

		archived := closedPath
		if sizeExceeded && !dayChanged {
			archived = archiveBySequence(closedPath, l.config.FilePrefix, closedDay)
		}
		if l.config.CompressRotated {
			_ = compressLogFile(archived)
		}
	}

	if err := os.MkdirAll(l.config.LogsDir, 0o755); err != nil {
		l.rotateErr = NewLogDirCreateError(l.config.LogsDir, err)
		return l.rotateErr
	}

	l.cleanupOldLogs()

	filename := l.config.FilePrefix + "-" + today + ".log"
	logPath := filepath.Join(l.config.LogsDir, filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		l.rotateErr = NewLogFileOpenError(logPath, err)
		return l.rotateErr
	}

	var size int64
	if info, statErr := file.Stat(); statErr == nil {
		size = info.Size()
	}

	l.logFile = file
	l.currentDay = today
	l.currentSize = size
	l.rotateErr = nil
	return nil
}

func archiveBySequence(path, prefix, date string) string {
	dir := filepath.Dir(path)
	seq := nextRotateIndex(dir, prefix, date)
	dest := filepath.Join(dir, fmt.Sprintf("%s-%s.%d.log", prefix, date, seq))
	if err := os.Rename(path, dest); err != nil {
		return path
	}
	return dest
}

func nextRotateIndex(dir, prefix, date string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix+"-"+date) + `\.(\d+)\.log(?:\.gz)?$`)
	max := 0
	for _, e := range entries {
		m := pattern.FindStringSubmatch(e.Name())
		if len(m) < 2 {
			continue
		}
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1
}

func compressLogFile(path string) error {
	if path == "" || strings.HasSuffix(path, ".gz") {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}

	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	outPath := path + ".gz"
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(out)
	_, copyErr := io.Copy(gz, in)
	closeGzErr := gz.Close()
	closeOutErr := out.Close()
	if copyErr != nil || closeGzErr != nil || closeOutErr != nil {
		_ = os.Remove(outPath)
		if copyErr != nil {
			return copyErr
		}
		if closeGzErr != nil {
			return closeGzErr
		}
		return closeOutErr
	}
	return os.Remove(path)
}

type archivedLog struct {
	path string
	date string
	seq  int
}

func (l *StructuredLogger) cleanupOldLogs() {
	if l.config.RetentionDays <= 0 && l.config.MaxBackups <= 0 {
		return
	}
	dir := l.config.LogsDir
	prefix := l.config.FilePrefix
	if prefix == "" {
		prefix = DefaultConfig().FilePrefix
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-(\d{4}-\d{2}-\d{2})(?:\.(\d+))?\.log(?:\.gz)?$`)
	today := time.Now().Format("2006-01-02")
	activeName := prefix + "-" + today + ".log"

	var files []archivedLog
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		m := pattern.FindStringSubmatch(name)
		if len(m) < 2 {
			continue
		}
		if name == activeName {
			continue
		}
		seq := 0
		if m[2] != "" {
			seq, _ = strconv.Atoi(m[2])
		}
		files = append(files, archivedLog{
			path: filepath.Join(dir, name),
			date: m[1],
			seq:  seq,
		})
	}

	if l.config.RetentionDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -l.config.RetentionDays)
		cutoffDate := cutoff.Format("2006-01-02")
		var kept []archivedLog
		for _, f := range files {
			if f.date < cutoffDate {
				_ = os.Remove(f.path)
				continue
			}
			kept = append(kept, f)
		}
		files = kept
	}

	if l.config.MaxBackups > 0 && len(files) > l.config.MaxBackups {
		sort.Slice(files, func(i, j int) bool {
			if files[i].date != files[j].date {
				return files[i].date > files[j].date
			}
			return files[i].seq > files[j].seq
		})
		for _, f := range files[l.config.MaxBackups:] {
			_ = os.Remove(f.path)
		}
	}
}
