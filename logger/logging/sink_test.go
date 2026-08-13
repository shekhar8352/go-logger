package logging

import (
	"context"
	"strings"
	"testing"
)

func TestLogger_WithBufferSink_WritesToBuffer(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.MinLevel = LevelInfo

	l := NewLogger(cfg)
	defer l.Close()

	if l.RotationError() != nil {
		t.Fatalf("did not expect rotation error with BufferSink: %v", l.RotationError())
	}

	l.Info(context.Background(), "hello from buffer sink")

	got := sink.String()
	if !strings.Contains(got, "hello from buffer sink") {
		t.Fatalf("expected buffer to contain log message, got %q", got)
	}
	if !strings.Contains(got, `"level":"INFO"`) {
		t.Fatalf("expected JSON INFO entry in buffer, got %q", got)
	}
}

func TestBufferSink_ReturnsWrittenLines(t *testing.T) {
	sink := NewBufferSink()

	if lines := sink.Lines(); lines != nil {
		t.Fatalf("expected nil lines on empty sink, got %v", lines)
	}

	if _, err := sink.Write([]byte("first line\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := sink.Write([]byte("second line\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := sink.Write([]byte("third line")); err != nil {
		t.Fatalf("write: %v", err)
	}

	lines := sink.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "first line" || lines[1] != "second line" || lines[2] != "third line" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}
