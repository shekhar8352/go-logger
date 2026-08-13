package logging

import (
	"os"
	"path/filepath"
	"time"
)

// rotateLogFile handles creating/rotating a log file for the given logger.
// On failure it leaves logFile nil so subsequent writes fall back to stderr
// and later calls can retry. It never panics.
func (l *StructuredLogger) rotateLogFile() error {
	today := time.Now().Format("2006-01-02")

	// if already current, nothing to do
	if today == l.currentDay && l.logFile != nil {
		return nil
	}

	// close existing
	if l.logFile != nil {
		_ = l.logFile.Close()
		l.logFile = nil
	}

	// ensure dir
	if err := os.MkdirAll(l.config.LogsDir, 0o755); err != nil {
		l.rotateErr = NewLogDirCreateError(l.config.LogsDir, err)
		return l.rotateErr
	}

	filename := l.config.FilePrefix + "-" + today + ".log"
	logPath := filepath.Join(l.config.LogsDir, filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		l.rotateErr = NewLogFileOpenError(logPath, err)
		return l.rotateErr
	}

	l.logFile = file
	l.currentDay = today
	l.rotateErr = nil
	return nil
}
