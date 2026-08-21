package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ReadLogsOptions provides configuration options for log reading operations
type ReadLogsOptions struct {
	MaxResults             int
	ChunkSize              int
	EnableEarlyTermination bool
}

// ReadAndFilterLogsWithOptions reads and filters logs from a file with performance optimizations
func ReadAndFilterLogsWithOptions(filename string, filter func(LogEntry) bool, options ReadLogsOptions) ([]LogEntry, int, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, 0, NewLogFileNotFoundError(filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, NewLogFileReadError(filename, fmt.Errorf("failed to open file: %w", err))
	}
	defer func() { _ = file.Close() }()

	chunkSize := options.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	var allLogs []LogEntry
	var skippedLines int
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentChunk []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		currentChunk = append(currentChunk, line)
		if len(currentChunk) >= chunkSize {
			processed, skipped := processLogChunk(currentChunk, filter)
			allLogs = append(allLogs, processed...)
			skippedLines += skipped
			currentChunk = currentChunk[:0]
			if options.EnableEarlyTermination && options.MaxResults > 0 && len(allLogs) >= options.MaxResults {
				if len(allLogs) > options.MaxResults {
					allLogs = allLogs[:options.MaxResults]
				}
				break
			}
		}
	}
	if len(currentChunk) > 0 {
		processed, skipped := processLogChunk(currentChunk, filter)
		allLogs = append(allLogs, processed...)
		skippedLines += skipped
		if options.EnableEarlyTermination && options.MaxResults > 0 && len(allLogs) > options.MaxResults {
			allLogs = allLogs[:options.MaxResults]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, NewLogFileReadError(filename, fmt.Errorf("error reading file: %w", err))
	}

	return allLogs, len(allLogs), nil
}

// ReadAndFilterLogs convenience wrapper
func ReadAndFilterLogs(filename string, filter func(LogEntry) bool) ([]LogEntry, int, error) {
	return ReadAndFilterLogsWithOptions(filename, filter, ReadLogsOptions{})
}

// ReadLogsByDate reads logs for a date from the default directory and prefix
// (GetLogDirectory() and "app"). See ReadLogsByDateWithConfig to use a custom path.
func ReadLogsByDate(date, level string) ([]LogEntry, int, error) {
	return ReadLogsByDateWithConfig(LoggerConfig{}, date, level)
}

// ReadLogsByDateWithConfig reads logs for a date using cfg.LogsDir and cfg.FilePrefix.
// Empty LogsDir falls back to GetLogDirectory(); empty FilePrefix falls back to "app".
func ReadLogsByDateWithConfig(cfg LoggerConfig, date, level string) ([]LogEntry, int, error) {
	return ReadLogsWithOptions(ReadOptions{Config: cfg, Date: date, Level: level})
}

// ReadLogsWithOptions reads logs for a date with optional level, time range, and pagination.
// The returned count is the number of matches before Offset/Limit are applied.
func ReadLogsWithOptions(opts ReadOptions) ([]LogEntry, int, error) {
	dir, prefix := readerDirAndPrefix(opts.Config)
	logFile := datedLogFile(dir, prefix, opts.Date)
	options := ReadLogsOptions{MaxResults: 0, ChunkSize: 1000, EnableEarlyTermination: false}
	logs, _, err := ReadAndFilterLogsWithOptions(logFile, func(e LogEntry) bool {
		if opts.Level != "" && !strings.EqualFold(e.Level, opts.Level) {
			return false
		}
		return entryInTimeRange(e, opts.StartTime, opts.EndTime)
	}, options)
	if err != nil {
		return nil, 0, err
	}
	total := len(logs)
	return applyPagination(logs, opts.Offset, opts.Limit), total, nil
}

// SearchOptions configures SearchLogsWithOptions. Existing SearchLogs helpers
// keep substring matching and no pagination.
type SearchOptions struct {
	Config    LoggerConfig
	Query     string
	Date      string
	Level     string
	Offset    int
	Limit     int
	StartTime time.Time
	EndTime   time.Time
	UseRegex  bool
}

// ReadOptions configures ReadLogsWithOptions.
type ReadOptions struct {
	Config    LoggerConfig
	Date      string
	Level     string
	Offset    int
	Limit     int
	StartTime time.Time
	EndTime   time.Time
}

// SearchLogs searches recent files (or a specific date) for a query using the
// default directory and prefix (GetLogDirectory() and "app"). See SearchLogsWithConfig
// to use a custom LogsDir and FilePrefix.
func SearchLogs(query, date, level string) ([]LogEntry, int, error) {
	return SearchLogsWithConfig(LoggerConfig{}, query, date, level)
}

// SearchLogsWithConfig searches recent files (or a specific date) for a query
// using cfg.LogsDir and cfg.FilePrefix. Empty fields fall back to GetLogDirectory()
// and "app" so behavior matches SearchLogs when no config is provided.
func SearchLogsWithConfig(cfg LoggerConfig, query, date, level string) ([]LogEntry, int, error) {
	return SearchLogsWithOptions(SearchOptions{Config: cfg, Query: query, Date: date, Level: level})
}

// SearchLogsWithOptions searches logs with pagination, time range, and optional regex.
// The returned count is the number of matches before Offset/Limit are applied.
func SearchLogsWithOptions(opts SearchOptions) ([]LogEntry, int, error) {
	var compiled *regexp.Regexp
	if opts.UseRegex && opts.Query != "" {
		re, err := regexp.Compile(opts.Query)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid search regex: %w", err)
		}
		compiled = re
	}

	dir, prefix := readerDirAndPrefix(opts.Config)
	var logFiles []string
	if opts.Date != "" {
		logFiles = []string{datedLogFile(dir, prefix, opts.Date)}
	} else {
		for i := 0; i < 7; i++ {
			d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			logFiles = append(logFiles, datedLogFile(dir, prefix, d))
		}
	}

	var allLogs []LogEntry
	for _, f := range logFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}
		options := ReadLogsOptions{MaxResults: 0, ChunkSize: 800, EnableEarlyTermination: false}
		logs, _, err := ReadAndFilterLogsWithOptions(f, func(e LogEntry) bool {
			if !matchSearchQuery(e, opts.Query, compiled) {
				return false
			}
			if opts.Level != "" && !strings.EqualFold(e.Level, opts.Level) {
				return false
			}
			return entryInTimeRange(e, opts.StartTime, opts.EndTime)
		}, options)
		if err != nil {
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	sort.Slice(allLogs, func(i, j int) bool { return allLogs[i].Timestamp > allLogs[j].Timestamp })
	total := len(allLogs)
	return applyPagination(allLogs, opts.Offset, opts.Limit), total, nil
}

// GetLogDirectory returns log dir path (env LOG_DIR or ./logs)
func GetLogDirectory() string {
	if d := os.Getenv("LOG_DIR"); d != "" {
		return d
	}
	return "./logs"
}

// GetDateRange returns inclusive date strings between start and end
func GetDateRange(startDate, endDate string) []string {
	var dates []string
	if startDate == "" && endDate == "" {
		for i := 6; i >= 0; i-- {
			d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			dates = append(dates, d)
		}
		return dates
	}
	start, err1 := time.Parse("2006-01-02", startDate)
	end, err2 := time.Parse("2006-01-02", endDate)
	if err1 != nil || err2 != nil {
		return []string{time.Now().Format("2006-01-02")}
	}
	if end.Before(start) {
		start, end = end, start
	}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}

// GetLast3Days returns last 3 days inclusive
func GetLast3Days() []string {
	var dates []string
	for i := 2; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dates = append(dates, d)
	}
	return dates
}

// GetAvailableLogFiles lists files like app-YYYY-MM-DD.log in the default
// directory, sorted newest first. See GetAvailableLogFilesWithConfig for a
// custom LogsDir and FilePrefix.
func GetAvailableLogFiles() ([]string, error) {
	return GetAvailableLogFilesWithConfig(LoggerConfig{})
}

// GetAvailableLogFilesWithConfig lists dated log files matching cfg.FilePrefix
// in cfg.LogsDir, sorted newest first. Empty fields fall back to GetLogDirectory()
// and "app".
func GetAvailableLogFilesWithConfig(cfg LoggerConfig) ([]string, error) {
	dir, prefix := readerDirAndPrefix(cfg)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, NewLogFileReadError(dir, fmt.Errorf("log directory does not exist"))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, NewLogFileReadError(dir, fmt.Errorf("failed to read log directory: %w", err))
	}
	pattern := logFileNamePattern(prefix)
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if pattern.MatchString(name) {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Slice(files, func(i, j int) bool {
		iDate := extractDateFromLogFile(files[i])
		jDate := extractDateFromLogFile(files[j])
		return iDate > jDate
	})
	return files, nil
}

// readerDirAndPrefix resolves LogsDir and FilePrefix from cfg, falling back to
// GetLogDirectory() and DefaultConfig().FilePrefix ("app") when empty.
func readerDirAndPrefix(cfg LoggerConfig) (dir, prefix string) {
	dir = cfg.LogsDir
	prefix = cfg.FilePrefix
	if dir == "" {
		dir = GetLogDirectory()
	}
	if prefix == "" {
		prefix = DefaultConfig().FilePrefix
	}
	return dir, prefix
}

func datedLogFile(dir, prefix, date string) string {
	return filepath.Join(dir, fmt.Sprintf("%s-%s.log", prefix, date))
}

func logFileNamePattern(prefix string) *regexp.Regexp {
	return regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-(\d{4}-\d{2}-\d{2})\.log$`)
}

var logFileDatePattern = regexp.MustCompile(`-(\d{4}-\d{2}-\d{2})\.log$`)

func extractDateFromLogFile(filePath string) string {
	name := filepath.Base(filePath)
	m := logFileDatePattern.FindStringSubmatch(name)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func matchSearchQuery(e LogEntry, query string, compiled *regexp.Regexp) bool {
	if compiled != nil {
		return compiled.MatchString(e.Message) || compiled.MatchString(e.Path) || compiled.MatchString(e.LogID)
	}
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(e.Message), q) ||
		strings.Contains(strings.ToLower(e.Path), q) ||
		strings.Contains(strings.ToLower(e.LogID), q)
}

func entryInTimeRange(e LogEntry, start, end time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return true
	}
	ts, err := parseEntryTimestamp(e.Timestamp)
	if err != nil {
		return false
	}
	if !start.IsZero() && ts.Before(start) {
		return false
	}
	if !end.IsZero() && ts.After(end) {
		return false
	}
	return true
}

func parseEntryTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.UnixMilli(ms), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

func applyPagination(logs []LogEntry, offset, limit int) []LogEntry {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(logs) {
		if logs == nil {
			return nil
		}
		return []LogEntry{}
	}
	logs = logs[offset:]
	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}
	return logs
}
