package logging

import (
	"context"

	"github.com/google/uuid"
)

// Context key type and key
type ContextKey string

const (
	LogIDKey     ContextKey = "log_id"
	RequestIDKey ContextKey = "request_id"
	TraceIDKey   ContextKey = "trace_id"
	UserIDKey    ContextKey = "user_id"
)

type contextFieldsKey struct{}

// AddLogID returns a new context with a generated log id when none is set.
// If a log id is already present (for example via WithLogID), ctx is returned unchanged.
func AddLogID(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if GetLogID(ctx) != "" {
		return ctx
	}
	return context.WithValue(ctx, LogIDKey, uuid.New().String())
}

// WithLogID returns a new context with the given log id, overwriting any existing value.
// Use this to propagate an incoming request header or other external identifier.
func WithLogID(ctx context.Context, id string) context.Context {
	return withContextString(ctx, LogIDKey, id)
}

// GetLogID extracts log id (empty string if not present)
func GetLogID(ctx context.Context) string {
	return contextString(ctx, LogIDKey)
}

// WithRequestID stores a request id on the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return withContextString(ctx, RequestIDKey, id)
}

// GetRequestID extracts the request id (empty string if not present).
func GetRequestID(ctx context.Context) string {
	return contextString(ctx, RequestIDKey)
}

// WithTraceID stores a trace id on the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return withContextString(ctx, TraceIDKey, id)
}

// GetTraceID extracts the trace id (empty string if not present).
func GetTraceID(ctx context.Context) string {
	return contextString(ctx, TraceIDKey)
}

// WithUserID stores a user id on the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return withContextString(ctx, UserIDKey, id)
}

// GetUserID extracts the user id (empty string if not present).
func GetUserID(ctx context.Context) string {
	return contextString(ctx, UserIDKey)
}

func withContextString(ctx context.Context, key ContextKey, value string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, value)
}

func contextString(ctx context.Context, key ContextKey) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(key).(string); ok {
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
