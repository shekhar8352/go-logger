package logging

import "fmt"

// Custom error constructors to give better context when reading files

func NewLogFileNotFoundError(path string) error {
	return fmt.Errorf("log file not found: %s", path)
}

func NewLogFileReadError(path string, err error) error {
	return fmt.Errorf("failed to read log file %s: %w", path, err)
}

func NewLogDirCreateError(dir string, err error) error {
	return fmt.Errorf("failed to create logs directory %s: %w", dir, err)
}

func NewLogFileOpenError(path string, err error) error {
	return fmt.Errorf("failed to open log file %s: %w", path, err)
}
