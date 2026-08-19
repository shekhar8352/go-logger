package logging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWithLogID_OverridesLogID(t *testing.T) {
	ctx := AddLogID(context.Background())
	generated := GetLogID(ctx)
	if generated == "" {
		t.Fatal("expected AddLogID to set a log id")
	}

	ctx = WithLogID(ctx, "incoming-from-header")
	if got := GetLogID(ctx); got != "incoming-from-header" {
		t.Fatalf("GetLogID() = %q, want %q", got, "incoming-from-header")
	}

	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	lg := NewLogger(cfg)
	defer lg.Close()

	lg.Info(ctx, "using external log id")
	out := sink.String()
	if !strings.Contains(out, `"log_id":"incoming-from-header"`) {
		t.Fatalf("expected overridden log id in output, got %s", out)
	}
	if strings.Contains(out, generated) {
		t.Fatalf("output should not contain the original generated id %q: %s", generated, out)
	}
}

func TestGetLogID_FromContext(t *testing.T) {
	if got := GetLogID(nil); got != "" {
		t.Fatalf("GetLogID(nil) = %q, want empty", got)
	}
	if got := GetLogID(context.Background()); got != "" {
		t.Fatalf("GetLogID(empty ctx) = %q, want empty", got)
	}

	ctx := WithLogID(context.Background(), "abc-123")
	if got := GetLogID(ctx); got != "abc-123" {
		t.Fatalf("GetLogID() = %q, want %q", got, "abc-123")
	}
}

func TestLogEntry_IncludesRequestIdWhenSet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Writer = NewBufferSink()
	cfg.IncludeConsole = false
	lg := NewLogger(cfg)
	defer lg.Close()

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = WithTraceID(ctx, "tr-1")
	ctx = WithUserID(ctx, "user-9")

	entry := lg.createLogEntry(ctx, LevelInfo, "traced")
	if entry.RequestID != "req-1" {
		t.Fatalf("RequestID = %q, want req-1", entry.RequestID)
	}
	if entry.TraceID != "tr-1" {
		t.Fatalf("TraceID = %q, want tr-1", entry.TraceID)
	}
	if entry.UserID != "user-9" {
		t.Fatalf("UserID = %q, want user-9", entry.UserID)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)
	for _, want := range []string{`"request_id":"req-1"`, `"trace_id":"tr-1"`, `"user_id":"user-9"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("JSON missing %s: %s", want, got)
		}
	}

	plain := lg.createLogEntry(context.Background(), LevelInfo, "no tracing")
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("marshal plain: %v", err)
	}
	for _, key := range []string{`"request_id"`, `"trace_id"`, `"user_id"`} {
		if strings.Contains(string(plainJSON), key) {
			t.Fatalf("unset tracing keys should be omitted, got %s", plainJSON)
		}
	}

	parsed, err := ParseLogLine(got)
	if err != nil {
		t.Fatalf("ParseLogLine: %v", err)
	}
	if parsed.RequestID != "req-1" || parsed.TraceID != "tr-1" || parsed.UserID != "user-9" {
		t.Fatalf("parsed tracing ids = %+v", parsed)
	}
}

func TestAddLogID_WhenIdNotSet_GeneratesNew(t *testing.T) {
	ctx := AddLogID(context.Background())
	id := GetLogID(ctx)
	if id == "" {
		t.Fatal("AddLogID should generate a log id when none is set")
	}

	again := AddLogID(ctx)
	if got := GetLogID(again); got != id {
		t.Fatalf("AddLogID should keep an existing id, got %q want %q", got, id)
	}

	other := AddLogID(context.Background())
	if GetLogID(other) == id {
		t.Fatal("separate AddLogID calls should generate distinct ids")
	}
}
