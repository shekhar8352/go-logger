package logging

import (
	"context"

	"github.com/google/uuid"
)

// Context key type and key
type ContextKey string

const LogIDKey ContextKey = "log_id"

// AddLogID returns a new context with a generated log id
func AddLogID(ctx context.Context) context.Context {
	return context.WithValue(ctx, LogIDKey, uuid.New().String())
}

// GetLogID extracts log id (empty string if not present)
func GetLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(LogIDKey).(string); ok {
		return v
	}
	return ""
}
