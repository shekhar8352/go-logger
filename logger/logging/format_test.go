package logging

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testLoggerWithFormat(t *testing.T, format, tsFormat string) (*StructuredLogger, *BufferSink) {
	t.Helper()
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.LogFormat = format
	if tsFormat != "" {
		cfg.TimestampFormat = tsFormat
	}
	return NewLogger(cfg), sink
}

func TestLogger_LogFormat_JSON(t *testing.T) {
	lg, sink := testLoggerWithFormat(t, FormatJSON, "")
	defer lg.Close()

	lg.Info(context.Background(), "json format message")
	line := strings.TrimSpace(sink.String())
	if line == "" {
		t.Fatal("expected JSON output")
	}
	if line[0] != '{' {
		t.Fatalf("expected JSON object, got %q", line)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v (%s)", err, line)
	}
	if raw["level"] != LevelInfo {
		t.Fatalf("level = %#v", raw["level"])
	}
	if raw["message"] != "json format message" {
		t.Fatalf("message = %#v", raw["message"])
	}
}

func TestLogger_LogFormat_Logfmt(t *testing.T) {
	lg, sink := testLoggerWithFormat(t, FormatLogfmt, "")
	defer lg.Close()

	lg.Info(context.Background(), "hello world")
	line := strings.TrimSpace(sink.String())
	if line == "" {
		t.Fatal("expected logfmt output")
	}
	if strings.HasPrefix(line, "{") {
		t.Fatalf("logfmt should not be JSON, got %q", line)
	}
	for _, part := range []string{"timestamp=", "level=INFO", `message="hello world"`} {
		if !strings.Contains(line, part) {
			t.Fatalf("expected %q in logfmt line, got %q", part, line)
		}
	}
	if err := json.Unmarshal([]byte(line), new(map[string]interface{})); err == nil {
		t.Fatalf("logfmt line unexpectedly parsed as JSON: %q", line)
	}
}

func TestLogger_TimestampFormat_AppliesToEntry(t *testing.T) {
	t.Run("unix_ms", func(t *testing.T) {
		before := time.Now().UnixMilli()
		lg, sink := testLoggerWithFormat(t, FormatJSON, TimestampUnixMs)
		defer lg.Close()
		lg.Info(context.Background(), "unix ms")
		after := time.Now().UnixMilli()

		entry, err := ParseLogLine(strings.TrimSpace(sink.String()))
		if err != nil {
			t.Fatalf("ParseLogLine: %v", err)
		}
		ms, err := strconv.ParseInt(entry.Timestamp, 10, 64)
		if err != nil {
			t.Fatalf("timestamp %q is not unix milliseconds: %v", entry.Timestamp, err)
		}
		if ms < before-5 || ms > after+5 {
			t.Fatalf("unix ms timestamp %d out of range [%d, %d]", ms, before, after)
		}
	})

	t.Run("custom_layout", func(t *testing.T) {
		lg, sink := testLoggerWithFormat(t, FormatJSON, "2006-01-02")
		defer lg.Close()
		lg.Info(context.Background(), "date only")

		entry, err := ParseLogLine(strings.TrimSpace(sink.String()))
		if err != nil {
			t.Fatalf("ParseLogLine: %v", err)
		}
		want := time.Now().Format("2006-01-02")
		if entry.Timestamp != want {
			t.Fatalf("timestamp = %q, want %q", entry.Timestamp, want)
		}
	})

	t.Run("default_rfc3339", func(t *testing.T) {
		lg, sink := testLoggerWithFormat(t, "", "")
		defer lg.Close()
		lg.Info(context.Background(), "rfc3339")

		entry, err := ParseLogLine(strings.TrimSpace(sink.String()))
		if err != nil {
			t.Fatalf("ParseLogLine: %v", err)
		}
		if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
			t.Fatalf("default timestamp %q is not RFC3339: %v", entry.Timestamp, err)
		}
	})
}

func TestParseLogLine_StillWorksForJSON(t *testing.T) {
	lg, sink := testLoggerWithFormat(t, FormatJSON, "")
	defer lg.Close()

	ctx := WithLogID(context.Background(), "parse-json-id")
	lg.Info(ctx, "still parseable")

	line := strings.TrimSpace(sink.String())
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("ParseLogLine should still parse JSON output: %v (%s)", err, line)
	}
	if entry.Message != "still parseable" {
		t.Fatalf("message = %q", entry.Message)
	}
	if entry.Level != LevelInfo {
		t.Fatalf("level = %q", entry.Level)
	}
	if entry.LogID != "parse-json-id" {
		t.Fatalf("log_id = %q", entry.LogID)
	}
}
