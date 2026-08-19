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

	// Optional tracing identifiers. Omitted from JSON when empty.
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`

	// Fields holds structured key-value data serialized as a nested "fields" object.
	// It is omitted from JSON when empty so existing log shape is unchanged.
	Fields map[string]interface{} `json:"fields,omitempty"`

	// Parsed fields (optional)
	Method    string            `json:"method,omitempty"`
	Path      string            `json:"path,omitempty"`
	Status    int               `json:"status,omitempty"`
	Duration  string            `json:"duration,omitempty"`
	IP        string            `json:"ip,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// WithField returns a copy of the entry with key set in Fields.
func (e LogEntry) WithField(key string, value interface{}) LogEntry {
	return e.WithFields(map[string]interface{}{key: value})
}

// WithFields returns a copy of the entry with fields merged into Fields.
// Incoming keys overwrite existing ones.
func (e LogEntry) WithFields(fields map[string]interface{}) LogEntry {
	e.Fields = mergeFields(copyFields(e.Fields), copyFields(fields))
	return e
}

func copyFields(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeFields(base, extra map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
