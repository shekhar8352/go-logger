package logging

import (
	"bytes"
	"strings"
	"sync"
)

// BufferSink is an in-memory io.Writer that captures log output.
// It is intended for tests and callers that need to inspect written lines
// without touching the filesystem.
type BufferSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// NewBufferSink returns an empty BufferSink.
func NewBufferSink() *BufferSink {
	return &BufferSink{}
}

// Write appends p to the in-memory buffer.
func (s *BufferSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

// String returns the full captured output.
func (s *BufferSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// Bytes returns a copy of the captured output.
func (s *BufferSink) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.buf.Bytes()
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// Lines returns captured output split on newlines. A trailing newline does
// not produce an extra empty line.
func (s *BufferSink) Lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	content := s.buf.String()
	if content == "" {
		return nil
	}
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// Reset clears captured output.
func (s *BufferSink) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}
