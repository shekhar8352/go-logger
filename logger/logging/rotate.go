package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// rotateLogFile handles creating/rotating a log file for the given logger
func (l *StructuredLogger) rotateLogFile() {
	today := time.Now().Format("2006-01-02")

	// if already current, nothing to do
	if today == l.currentDay && l.logFile != nil {
		return
	}

	// close existing
	if l.logFile != nil {
		_ = l.logFile.Close()
	}

	// ensure dir
	if err := os.MkdirAll(l.config.LogsDir, 0o755); err != nil {
		panic(fmt.Sprintf("failed to create logs directory: %v", err))
	}

	filename := fmt.Sprintf("%s-%s.log", l.config.FilePrefix, today)
	logPath := filepath.Join(l.config.LogsDir, filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(fmt.Sprintf("failed to open log file: %v", err))
	}

	l.logFile = file
	l.currentDay = today
}
