package logging

import (
	"context"

	"github.com/google/uuid"
)

// Context key type and key
type ContextKey string

const LogIDKey ContextKey = "log_id"

type contextFieldsKey struct{}

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

// ContextWithField returns a context that carries an additional structured field.
// Existing context fields are copied; key overwrites a previous value.
func ContextWithField(ctx context.Context, key string, value interface{}) context.Context {
	return ContextWithFields(ctx, map[string]interface{}{key: value})
}

// ContextWithFields returns a context that carries structured fields.
// Incoming keys overwrite fields already stored on ctx.
func ContextWithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	merged := mergeFields(copyFields(GetContextFields(ctx)), copyFields(fields))
	return context.WithValue(ctx, contextFieldsKey{}, merged)
}

// GetContextFields returns a copy of structured fields stored on ctx.
// It returns nil when none are set.
func GetContextFields(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	fields, _ := ctx.Value(contextFieldsKey{}).(map[string]interface{})
	return copyFields(fields)
}
