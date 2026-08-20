package logging

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type delayWriter struct {
	delay time.Duration
	inner io.Writer
}

func (w *delayWriter) Write(p []byte) (int, error) {
	time.Sleep(w.delay)
	if w.inner != nil {
		return w.inner.Write(p)
	}
	return len(p), nil
}

func TestAsyncWriter_FlushesOnClose(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.UseAsyncWriter = true
	cfg.AsyncBufferSize = 8

	lg := NewLogger(cfg)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		lg.Info(ctx, "async-line-%d", i)
	}
	if err := lg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := sink.String()
	for i := 0; i < 5; i++ {
		want := "async-line-" + strconv.Itoa(i)
		if !strings.Contains(got, want) {
			t.Fatalf("after Close missing %q in %q", want, got)
		}
	}
}

func TestAsyncWriter_DoesNotBlock(t *testing.T) {
	sink := NewBufferSink()
	slow := &delayWriter{delay: 80 * time.Millisecond, inner: sink}
	cfg := DefaultConfig()
	cfg.Writer = slow
	cfg.IncludeConsole = false
	cfg.UseAsyncWriter = true
	cfg.AsyncBufferSize = 32

	lg := NewLogger(cfg)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 5; i++ {
		lg.Info(ctx, "non-blocking-%d", i)
	}
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Fatalf("async Info calls blocked for %v, want enqueue to return quickly", elapsed)
	}
	if err := lg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !strings.Contains(sink.String(), "non-blocking-4") {
		t.Fatalf("expected delayed writes to land after Close, got %q", sink.String())
	}
}

func TestLogger_StderrLevels_ErrorsToStderr(t *testing.T) {
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = wOut, wErr
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
	}()

	cfg := DefaultConfig()
	cfg.Writer = io.Discard
	cfg.IncludeConsole = true
	cfg.MinLevel = LevelDebug
	cfg.StderrLevels = []string{LevelError}

	lg := NewLogger(cfg)
	ctx := context.Background()
	lg.Info(ctx, "info-to-stdout")
	lg.Error(ctx, "error-to-stderr")
	if closeErr := lg.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	_ = wOut.Close()
	_ = wErr.Close()
	stdout, _ := io.ReadAll(rOut)
	stderr, _ := io.ReadAll(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	if !strings.Contains(string(stdout), "info-to-stdout") {
		t.Fatalf("stdout should contain info line, got %q", stdout)
	}
	if strings.Contains(string(stdout), "error-to-stderr") {
		t.Fatalf("stdout should not contain error line, got %q", stdout)
	}
	if !strings.Contains(string(stderr), "error-to-stderr") {
		t.Fatalf("stderr should contain error line, got %q", stderr)
	}
	if strings.Contains(string(stderr), "info-to-stdout") {
		t.Fatalf("stderr should not contain info line, got %q", stderr)
	}
}

func TestLogger_MultipleSinks_AllReceive(t *testing.T) {
	primary := NewBufferSink()
	a := NewBufferSink()
	b := NewBufferSink()

	cfg := DefaultConfig()
	cfg.Writer = primary
	cfg.IncludeConsole = false
	cfg.Sinks = []io.Writer{a, b}

	lg := NewLogger(cfg)
	defer lg.Close()
	lg.Info(context.Background(), "fan-out-message")

	for i, s := range []*BufferSink{primary, a, b} {
		if !strings.Contains(s.String(), "fan-out-message") {
			t.Fatalf("sink %d missing message, got %q", i, s.String())
		}
	}
}
