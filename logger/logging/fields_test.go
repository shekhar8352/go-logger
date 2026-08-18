package logging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogEntry_WithFields_SerializesToJSON(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2026-08-18T12:00:00Z",
		LogID:     "abc",
		Level:     LevelInfo,
		Message:   "hello",
		File:      "entry.go",
		Line:      10,
	}.WithField("user_id", 123).WithFields(map[string]interface{}{"role": "admin", "ok": true})

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"fields"`) {
		t.Fatalf("expected nested fields object, got %s", got)
	}
	if !strings.Contains(got, `"user_id":123`) {
		t.Fatalf("expected user_id in JSON, got %s", got)
	}
	if !strings.Contains(got, `"role":"admin"`) {
		t.Fatalf("expected role in JSON, got %s", got)
	}
	if !strings.Contains(got, `"ok":true`) {
		t.Fatalf("expected ok in JSON, got %s", got)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	fields, ok := raw["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("fields should be an object, got %#v", raw["fields"])
	}
	if fields["user_id"] != float64(123) {
		t.Fatalf("fields.user_id = %#v", fields["user_id"])
	}

	plain := LogEntry{Timestamp: "2026-08-18T12:00:00Z", Level: LevelInfo, Message: "plain"}
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("marshal plain: %v", err)
	}
	if strings.Contains(string(plainJSON), `"fields"`) {
		t.Fatalf("empty fields should be omitted, got %s", plainJSON)
	}
}

func TestLogger_WithFields_ChildAddsFieldsToEveryEntry(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.MinLevel = LevelDebug

	parent := NewLogger(cfg)
	defer parent.Close()
	child := parent.WithFields(map[string]interface{}{
		"user_id": 123,
		"role":    "admin",
	})

	ctx := context.Background()
	child.Debug(ctx, "debug msg")
	child.Info(ctx, "info msg")
	child.Warn(ctx, "warn msg")
	child.Error(ctx, "error msg")

	lines := sink.Lines()
	if len(lines) != 4 {
		t.Fatalf("expected 4 log lines, got %d: %v", len(lines), lines)
	}

	wantLevels := []string{LevelDebug, LevelInfo, LevelWarn, LevelError}
	for i, line := range lines {
		entry, err := ParseLogLine(line)
		if err != nil {
			t.Fatalf("parse line %d: %v", i, err)
		}
		if entry.Level != wantLevels[i] {
			t.Fatalf("line %d level = %q, want %q", i, entry.Level, wantLevels[i])
		}
		if entry.Fields["user_id"] != float64(123) {
			t.Fatalf("line %d missing user_id field: %#v", i, entry.Fields)
		}
		if entry.Fields["role"] != "admin" {
			t.Fatalf("line %d missing role field: %#v", i, entry.Fields)
		}
	}

	sink.Reset()
	parent.Info(ctx, "parent without child fields")
	parentOut := sink.String()
	if strings.Contains(parentOut, `"fields"`) {
		t.Fatalf("parent logger should not add child fields, got %s", parentOut)
	}
}

func TestCreateLogEntry_IncludesContextAndExtraFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IncludeConsole = false
	cfg.Writer = NewBufferSink()
	parent := NewLogger(cfg)
	defer parent.Close()
	child := parent.WithFields(map[string]interface{}{"service": "api"})

	ctx := AddLogID(context.Background())
	ctx = ContextWithField(ctx, "request_id", "req-1")

	entry := child.createLogEntry(ctx, LevelInfo, "hello %s", "world")
	if entry.LogID == "" {
		t.Fatal("expected log id from context")
	}
	if entry.Message != "hello world" {
		t.Fatalf("message = %q", entry.Message)
	}
	if entry.Fields["service"] != "api" {
		t.Fatalf("expected logger fields, got %#v", entry.Fields)
	}
	if entry.Fields["request_id"] != "req-1" {
		t.Fatalf("expected context fields, got %#v", entry.Fields)
	}

	// Context fields overlay logger fields on the same key.
	ctx = ContextWithField(ctx, "service", "gateway")
	overlaid := child.createLogEntry(ctx, LevelInfo, "overlay")
	if overlaid.Fields["service"] != "gateway" {
		t.Fatalf("context field should override logger field, got %#v", overlaid.Fields)
	}
}

func TestParser_ParseLogLine_WithExtraFields(t *testing.T) {
	line := `{"timestamp":"2026-08-18T12:00:00Z","log_id":"id-1","level":"INFO","message":"hello","file":"parser.go","line":1,"fields":{"user_id":123,"name":"ada"},"unknown":"keep-me"}`
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("ParseLogLine: %v", err)
	}
	if entry.Message != "hello" {
		t.Fatalf("message = %q", entry.Message)
	}
	if entry.Fields["user_id"] != float64(123) {
		t.Fatalf("fields.user_id = %#v", entry.Fields["user_id"])
	}
	if entry.Fields["name"] != "ada" {
		t.Fatalf("fields.name = %#v", entry.Fields["name"])
	}
	if entry.Extra["unknown"] != "keep-me" {
		t.Fatalf("extra.unknown = %#v", entry.Extra)
	}
	if _, ok := entry.Extra["fields"]; ok {
		t.Fatalf("fields object should not also land in Extra: %#v", entry.Extra)
	}

	plain := `{"timestamp":"2026-08-18T12:00:00Z","level":"INFO","message":"plain"}`
	parsed, err := ParseLogLine(plain)
	if err != nil {
		t.Fatalf("ParseLogLine plain: %v", err)
	}
	if parsed.Fields != nil {
		t.Fatalf("expected nil Fields without fields key, got %#v", parsed.Fields)
	}
}
