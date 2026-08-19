package logging

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func formatTimestamp(t time.Time, layout string) string {
	switch strings.TrimSpace(layout) {
	case "", time.RFC3339:
		return t.Format(time.RFC3339)
	case TimestampUnixMs, "unixmilli", "unix_milli":
		return strconv.FormatInt(t.UnixMilli(), 10)
	default:
		return t.Format(layout)
	}
}

func encodeLogLine(format string, entry LogEntry) []byte {
	if strings.EqualFold(strings.TrimSpace(format), FormatLogfmt) {
		return encodeLogfmt(entry)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return []byte(fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"json marshal error: %v"}`,
			entry.Timestamp, entry.Level, err))
	}
	return data
}

func encodeLogfmt(entry LogEntry) []byte {
	var b strings.Builder
	first := true
	appendKV := func(key, value string) {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		b.WriteString(key)
		b.WriteByte('=')
		if logfmtNeedsQuote(value) {
			b.WriteByte('"')
			b.WriteString(logfmtEscape(value))
			b.WriteByte('"')
		} else {
			b.WriteString(value)
		}
	}

	appendKV("timestamp", entry.Timestamp)
	appendKV("level", entry.Level)
	appendKV("message", entry.Message)
	appendKV("log_id", entry.LogID)
	if entry.RequestID != "" {
		appendKV("request_id", entry.RequestID)
	}
	if entry.TraceID != "" {
		appendKV("trace_id", entry.TraceID)
	}
	if entry.UserID != "" {
		appendKV("user_id", entry.UserID)
	}
	appendKV("file", entry.File)
	appendKV("line", strconv.Itoa(entry.Line))
	if entry.Function != "" {
		appendKV("function", entry.Function)
	}
	for k, v := range entry.Fields {
		appendKV(k, fmt.Sprint(v))
	}
	return []byte(b.String())
}

func logfmtNeedsQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == '=' || r == '"' || r == '\\' || unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func logfmtEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
