package logging

import (
	"context"
	"testing"
	"time"
)

func waitForHook(t *testing.T, ch <-chan LogEntry) LogEntry {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hook")
		return LogEntry{}
	}
}

func TestHook_InvokedWithEntry(t *testing.T) {
	ch := make(chan LogEntry, 2)
	cfg := DefaultConfig()
	cfg.Writer = NewBufferSink()
	cfg.IncludeConsole = false
	cfg.Hooks = []Hook{
		func(e LogEntry) { ch <- e },
	}
	cfg.OnError = func(e LogEntry) { ch <- e }

	lg := NewLogger(cfg)
	defer lg.Close()

	lg.Info(context.Background(), "hook-info")
	got := waitForHook(t, ch)
	if got.Level != LevelInfo || got.Message != "hook-info" {
		t.Fatalf("hook entry = %+v", got)
	}
	if lg.Metrics().Info != 1 {
		t.Fatalf("Info metric = %d, want 1", lg.Metrics().Info)
	}

	lg.Error(context.Background(), "hook-error")
	first := waitForHook(t, ch)
	second := waitForHook(t, ch)
	messages := map[string]string{first.Level: first.Message, second.Level: second.Message}
	if messages[LevelError] != "hook-error" {
		t.Fatalf("expected Hooks and OnError to observe ERROR, got %#v", messages)
	}
}

func TestHook_RespectsMinLevel(t *testing.T) {
	var seen []string
	done := make(chan struct{}, 1)
	cfg := DefaultConfig()
	cfg.Writer = NewBufferSink()
	cfg.IncludeConsole = false
	cfg.MinLevel = LevelError
	cfg.Hooks = []Hook{
		func(e LogEntry) {
			seen = append(seen, e.Level)
			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	lg := NewLogger(cfg)
	defer lg.Close()

	lg.Debug(context.Background(), "skip-debug")
	lg.Info(context.Background(), "skip-info")
	lg.Warn(context.Background(), "skip-warn")

	select {
	case <-done:
		t.Fatalf("hook ran for filtered-out levels: %v", seen)
	case <-time.After(50 * time.Millisecond):
	}

	m := lg.Metrics()
	if m.Debug != 0 || m.Info != 0 || m.Warn != 0 {
		t.Fatalf("metrics should ignore filtered levels, got %+v", m)
	}

	lg.Error(context.Background(), "keep-error")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hook was not invoked for ERROR")
	}
	if len(seen) != 1 || seen[0] != LevelError {
		t.Fatalf("hook levels = %v, want [ERROR]", seen)
	}
	if lg.Metrics().Error != 1 {
		t.Fatalf("Error metric = %d, want 1", lg.Metrics().Error)
	}
}

func TestHook_SlowHook_DoesNotBlockLogger(t *testing.T) {
	started := make(chan struct{})
	cfg := DefaultConfig()
	cfg.Writer = NewBufferSink()
	cfg.IncludeConsole = false
	cfg.Hooks = []Hook{
		func(LogEntry) {
			close(started)
			time.Sleep(200 * time.Millisecond)
		},
	}

	lg := NewLogger(cfg)
	defer lg.Close()

	start := time.Now()
	lg.Info(context.Background(), "slow-hook")
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Fatalf("logger blocked on slow hook for %v", elapsed)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("slow hook never started")
	}
}
