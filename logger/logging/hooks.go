package logging

// LogMetrics is a snapshot of how many entries were written per level
// after MinLevel filtering.
type LogMetrics struct {
	Debug uint64
	Info  uint64
	Warn  uint64
	Error uint64
}

// Metrics returns counters for entries that were actually logged.
func (l *StructuredLogger) Metrics() LogMetrics {
	w := l.rootOrSelf()
	return LogMetrics{
		Debug: w.debugCount.Load(),
		Info:  w.infoCount.Load(),
		Warn:  w.warnCount.Load(),
		Error: w.errorCount.Load(),
	}
}

func (l *StructuredLogger) recordMetrics(level string) {
	switch level {
	case LevelDebug:
		l.debugCount.Add(1)
	case LevelInfo:
		l.infoCount.Add(1)
	case LevelWarn:
		l.warnCount.Add(1)
	case LevelError:
		l.errorCount.Add(1)
	}
}

func (l *StructuredLogger) invokeHooks(entry LogEntry) {
	hooks := l.config.Hooks
	onError := l.config.OnError
	if len(hooks) == 0 && onError == nil {
		return
	}
	go runHooks(hooks, onError, entry)
}

func runHooks(hooks []Hook, onError Hook, entry LogEntry) {
	for _, h := range hooks {
		callHook(h, entry)
	}
	if entry.Level == LevelError {
		callHook(onError, entry)
	}
}

func callHook(h Hook, entry LogEntry) {
	if h == nil {
		return
	}
	defer func() { _ = recover() }()
	h(entry)
}
