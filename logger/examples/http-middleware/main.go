package main

import (
	"context"
	"net/http"
	"time"

	"github.com/shekhar8352/go-logger/logger/logging"
)

// LoggingMiddleware injects a Log ID into the context and logs the request details
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 1. Compute/Inject Log ID
		ctx := logging.AddLogID(r.Context())

		// 2. Log Request Start
		logging.Info(ctx, "Started %s %s", r.Method, r.URL.Path)

		// 3. Pass new context with LogID to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))

		// 4. Log Request Completed with Duration
		duration := time.Since(start)
		logging.Info(ctx, "Completed %s %s in %v", r.Method, r.URL.Path, duration)
	})
}

// HelloHandler is an example handler
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// The logger will automatically pick up the LogID from the context
	logging.Debug(ctx, "Processing business logic for hello handler")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World! Check your logs."))
}

func main() {
	// Initialize default logger
	lg := logging.NewLogger(logging.DefaultConfig())
	defer lg.Close()

	// Use global default logger for simplicity in this example
	logging.DefaultLogger = lg

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", HelloHandler)

	// Wrap mux with our middleware
	loggedMux := LoggingMiddleware(mux)

	logging.Info(context.Background(), "Server starting on :8080")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		logging.Error(context.Background(), "Server failed: %v", err)
	}
}
