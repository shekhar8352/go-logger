package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// ReadLogsByDate convenience
func ReadLogsByDate(date, level string) ([]LogEntry, int, error) {
	logDir := GetLogDirectory()
	logFile := filepath.Join(logDir, fmt.Sprintf("app-%s.log", date))
	options := ReadLogsOptions{MaxResults: 0, ChunkSize: 1000, EnableEarlyTermination: false}
	return ReadAndFilterLogsWithOptions(logFile, func(e LogEntry) bool {
		if level != "" && !strings.EqualFold(e.Level, level) {
			return false
		}
		return true
	}, options)
}

// SearchLogs searches recent files (or specific date) for a query
func SearchLogs(query, date, level string) ([]LogEntry, int, error) {
	var logFiles []string
	logDir := GetLogDirectory()
	if date != "" {
		logFiles = []string{filepath.Join(logDir, fmt.Sprintf("app-%s.log", date))}
	} else {
		for i := 0; i < 7; i++ {
			d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			logFiles = append(logFiles, filepath.Join(logDir, fmt.Sprintf("app-%s.log", d)))
		}
	}
	var allLogs []LogEntry
	q := strings.ToLower(query)
	for _, f := range logFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}
		options := ReadLogsOptions{MaxResults: 5000, ChunkSize: 800, EnableEarlyTermination: true}
		logs, _, err := ReadAndFilterLogsWithOptions(f, func(e LogEntry) bool {
			if strings.Contains(strings.ToLower(e.Message), q) || strings.Contains(strings.ToLower(e.Path), q) || strings.Contains(strings.ToLower(e.LogID), q) {
				if level != "" && !strings.EqualFold(e.Level, level) {
					return false
				}
				return true
			}
			return false
		}, options)
		if err != nil {
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	sort.Slice(allLogs, func(i, j int) bool { return allLogs[i].Timestamp > allLogs[j].Timestamp })
	return allLogs, len(allLogs), nil
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

// GetAvailableLogFiles lists files like app-YYYY-MM-DD.log sorted newest first
func GetAvailableLogFiles() ([]string, error) {
	logDir := GetLogDirectory()
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return nil, NewLogFileReadError(logDir, fmt.Errorf("log directory does not exist"))
	}
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, NewLogFileReadError(logDir, fmt.Errorf("failed to read log directory: %w", err))
	}
	pattern := regexp.MustCompile(`^app-(\d{4}-\d{2}-\d{2})\.log$`)
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if pattern.MatchString(name) {
			files = append(files, filepath.Join(logDir, name))
		}
	}
	sort.Slice(files, func(i, j int) bool {
		iDate := extractDateFromLogFile(files[i])
		jDate := extractDateFromLogFile(files[j])
		return iDate > jDate
	})
	return files, nil
}

func extractDateFromLogFile(filePath string) string {
	name := filepath.Base(filePath)
	pattern := regexp.MustCompile(`^app-(\d{4}-\d{2}-\d{2})\.log$`)
	m := pattern.FindStringSubmatch(name)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
