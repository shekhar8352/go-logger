package logging

// LogEntry is the portable log structure used by both writer and reader.
// Timestamp is RFC3339 string for simplicity across writing/reading.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	LogID     string `json:"log_id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Function  string `json:"function,omitempty"`

	// Parsed fields (optional)
	Method    string            `json:"method,omitempty"`
	Path      string            `json:"path,omitempty"`
	Status    int               `json:"status,omitempty"`
	Duration  string            `json:"duration,omitempty"`
	IP        string            `json:"ip,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}
