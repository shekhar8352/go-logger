package logging

import (
	"strings"
	"sync/atomic"
)

func (l *StructuredLogger) dropBySampling(level string) bool {
	n := l.config.SampleEveryN
	if n <= 1 {
		return false
	}
	if !samplingAppliesTo(level, l.config.SampleLevels) {
		return false
	}
	c := l.sampleCounter(level).Add(1)
	return c%uint64(n) != 1
}

func samplingAppliesTo(level string, levels []string) bool {
	if len(levels) == 0 {
		return strings.EqualFold(level, LevelDebug)
	}
	for _, l := range levels {
		if strings.EqualFold(l, level) {
			return true
		}
	}
	return false
}

func (l *StructuredLogger) sampleCounter(level string) *atomic.Uint64 {
	switch strings.ToUpper(level) {
	case LevelDebug:
		return &l.sampleDebug
	case LevelInfo:
		return &l.sampleInfo
	case LevelWarn:
		return &l.sampleWarn
	case LevelError:
		return &l.sampleError
	default:
		return &l.sampleInfo
	}
}
