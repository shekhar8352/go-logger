package logging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestError_WithStackTrace_IncludesStack(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.IncludeStackTraceOnError = true

	lg := NewLogger(cfg)
	defer lg.Close()

	lg.Error(context.Background(), "boom")
	lg.Info(context.Background(), "not an error")

	lines := sink.Lines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}

	errEntry, err := ParseLogLine(lines[0])
	if err != nil {
		t.Fatalf("parse error line: %v", err)
	}
	if errEntry.StackTrace == "" {
		t.Fatal("expected non-empty stack_trace on ERROR")
	}
	if !strings.Contains(errEntry.StackTrace, "goroutine") {
		t.Fatalf("stack_trace should look like a Go stack, got %q", errEntry.StackTrace)
	}
	if !strings.Contains(errEntry.StackTrace, "TestError_WithStackTrace_IncludesStack") {
		t.Fatalf("stack_trace should include the test caller, got %q", errEntry.StackTrace)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &raw); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := raw["stack_trace"]; !ok {
		t.Fatalf("JSON missing stack_trace: %s", lines[0])
	}

	infoEntry, err := ParseLogLine(lines[1])
	if err != nil {
		t.Fatalf("parse info line: %v", err)
	}
	if infoEntry.StackTrace != "" {
		t.Fatalf("INFO should not include a stack trace, got %q", infoEntry.StackTrace)
	}
}

func TestError_WithoutStackTrace_NoStackField(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.IncludeStackTraceOnError = false

	lg := NewLogger(cfg)
	defer lg.Close()
	lg.Error(context.Background(), "no stack")

	line := strings.TrimSpace(sink.String())
	if strings.Contains(line, "stack_trace") {
		t.Fatalf("stack_trace should be omitted when disabled, got %s", line)
	}

	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("ParseLogLine: %v", err)
	}
	if entry.StackTrace != "" {
		t.Fatalf("parsed StackTrace = %q, want empty", entry.StackTrace)
	}
}
