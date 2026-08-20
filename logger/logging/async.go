package logging

import "time"

func (l *StructuredLogger) asyncLoop() {
	defer close(l.asyncDone)

	var tickerC <-chan time.Time
	if l.config.FlushInterval > 0 {
		ticker := time.NewTicker(l.config.FlushInterval)
		defer ticker.Stop()
		tickerC = ticker.C
	}

	for {
		select {
		case job, ok := <-l.asyncCh:
			if !ok {
				l.flushOutputs()
				return
			}
			l.mu.Lock()
			l.emitLocked(job)
			l.mu.Unlock()
		case <-tickerC:
			l.flushOutputs()
		}
	}
}

func (l *StructuredLogger) flushOutputs() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		_ = l.logFile.Sync()
	}
	flushWriter(l.config.Writer)
	for _, s := range l.config.Sinks {
		flushWriter(s)
	}
}

type flusher interface {
	Flush() error
}

type syncer interface {
	Sync() error
}

func flushWriter(w interface{}) {
	if w == nil {
		return
	}
	if f, ok := w.(flusher); ok {
		_ = f.Flush()
		return
	}
	if s, ok := w.(syncer); ok {
		_ = s.Sync()
	}
}
