package logging

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseLogLine parses a JSON log line into LogEntry. It tolerates missing/extra fields.
func ParseLogLine(line string) (LogEntry, error) {
	// Generic JSON map to decode unknown extras
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogEntry{}, fmt.Errorf("failed to parse JSON log line: %w", err)
	}

	entry := LogEntry{Extra: map[string]string{}}

	if ts, ok := raw["timestamp"].(string); ok {
		entry.Timestamp = ts
	}
	if id, ok := raw["log_id"].(string); ok {
		entry.LogID = id
	}
	if lvl, ok := raw["level"].(string); ok {
		entry.Level = lvl
	}
	if msg, ok := raw["message"].(string); ok {
		entry.Message = msg
	}
	if f, ok := raw["file"].(string); ok {
		entry.File = f
	}
	if ln, ok := raw["line"].(float64); ok {
		entry.Line = int(ln)
	}
	if fn, ok := raw["function"].(string); ok {
		entry.Function = fn
	}
	if id, ok := raw["request_id"].(string); ok {
		entry.RequestID = id
	}
	if id, ok := raw["trace_id"].(string); ok {
		entry.TraceID = id
	}
	if id, ok := raw["user_id"].(string); ok {
		entry.UserID = id
	}
	if st, ok := raw["stack_trace"].(string); ok {
		entry.StackTrace = st
	}
	if f, ok := raw["fields"].(map[string]interface{}); ok && len(f) > 0 {
		entry.Fields = copyFields(f)
	}

	// Capture any non-standard fields into Extra
	for k, v := range raw {
		switch k {
		case "timestamp", "log_id", "level", "message", "file", "line", "function",
			"request_id", "trace_id", "user_id", "fields", "stack_trace":
		// already handled
		default:
			if s, ok := v.(string); ok {
				entry.Extra[k] = s
			} else if v != nil {
				entry.Extra[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Backwards-compat: try to extract HTTP details from message
	message := entry.Message
	if httpMatch := regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+([^\s\|]+)`).FindStringSubmatch(message); len(httpMatch) > 2 {
		entry.Method = httpMatch[1]
		entry.Path = httpMatch[2]
	}
	if statusMatch := regexp.MustCompile(`Status:\s*(\d+)`).FindStringSubmatch(message); len(statusMatch) > 1 {
		if st, err := strconv.Atoi(statusMatch[1]); err == nil {
			entry.Status = st
		}
	}
	if durationMatch := regexp.MustCompile(`Duration:\s*([^\s\|]+)`).FindStringSubmatch(message); len(durationMatch) > 1 {
		entry.Duration = durationMatch[1]
	}
	if ipMatch := regexp.MustCompile(`IP:\s*([^\s\|]+)`).FindStringSubmatch(message); len(ipMatch) > 1 {
		entry.IP = ipMatch[1]
	}
	if uaMatch := regexp.MustCompile(`User-Agent:\s*([^|\r\n]+)`).FindStringSubmatch(message); len(uaMatch) > 1 {
		entry.UserAgent = strings.TrimSpace(uaMatch[1])
	}

	return entry, nil
}

// processLogChunk converts lines to log entries applying optional filter
func processLogChunk(lines []string, filter func(LogEntry) bool) ([]LogEntry, int) {
	var logs []LogEntry
	var skipped int
	for _, ln := range lines {
		if ln == "" {
			continue
		}
		entry, err := ParseLogLine(ln)
		if err != nil {
			skipped++
			continue
		}
		if filter == nil || filter(entry) {
			logs = append(logs, entry)
		}
	}
	return logs, skipped
}
