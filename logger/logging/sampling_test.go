package logging

import (
	"context"
	"strings"
	"testing"
)

func TestLogger_WithSampling_ReducesVolume(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.MinLevel = LevelDebug
	cfg.SampleEveryN = 5
	cfg.SampleLevels = []string{LevelDebug}

	lg := NewLogger(cfg)
	defer lg.Close()

	ctx := context.Background()
	const n = 20
	for i := 0; i < n; i++ {
		lg.Debug(ctx, "sampled-debug-%d", i)
		lg.Info(ctx, "unsampled-info-%d", i)
	}

	var debugLines, infoLines int
	for _, line := range sink.Lines() {
		if strings.Contains(line, "sampled-debug-") {
			debugLines++
		}
		if strings.Contains(line, "unsampled-info-") {
			infoLines++
		}
	}
	// 1-in-5 of 20 DEBUG lines → 4 kept (1st, 6th, 11th, 16th)
	if debugLines != 4 {
		t.Fatalf("sampled DEBUG count = %d, want 4", debugLines)
	}
	if infoLines != n {
		t.Fatalf("INFO should not be sampled, got %d want %d", infoLines, n)
	}
	if lg.Metrics().Debug != 4 {
		t.Fatalf("Debug metric = %d, want 4 (sampled writes only)", lg.Metrics().Debug)
	}
}

func TestLogger_WithoutSampling_LogsAll(t *testing.T) {
	sink := NewBufferSink()
	cfg := DefaultConfig()
	cfg.Writer = sink
	cfg.IncludeConsole = false
	cfg.MinLevel = LevelDebug
	cfg.SampleEveryN = 0

	lg := NewLogger(cfg)
	defer lg.Close()

	ctx := context.Background()
	const n = 10
	for i := 0; i < n; i++ {
		lg.Debug(ctx, "all-debug-%d", i)
		lg.Info(ctx, "all-info-%d", i)
	}

	lines := sink.Lines()
	if len(lines) != n*2 {
		t.Fatalf("without sampling expected %d lines, got %d", n*2, len(lines))
	}
	m := lg.Metrics()
	if m.Debug != uint64(n) || m.Info != uint64(n) {
		t.Fatalf("metrics = %+v, want Debug=%d Info=%d", m, n, n)
	}
}
