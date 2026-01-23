package main

import (
	"context"
	"time"

	"github.com/shekhar8352/go-logger/logger/logging"
)

func main() {
	// 1. Initialize variables
	// DefaultConfig logs to ./logs directory
	config := logging.DefaultConfig()
	config.IncludeConsole = true // Ensure we see output in console
	lg := logging.NewLogger(config)
	defer lg.Close()

	// 2. Create a context with a Log ID
	// This ID will be attached to all logs using this context, allowing for request tracing
	ctx := logging.AddLogID(context.Background())

	// 3. Log some messages
	lg.Info(ctx, "Starting the basic example application...")

	// Simulate some work
	doWork(ctx, lg)

	lg.Info(ctx, "Application finished successfully")

	// 4. Demonstrate error logging
	err := simulateError()
	if err != nil {
		lg.Error(ctx, "An unexpected error occurred: %v", err)
	}
}

func doWork(ctx context.Context, lg *logging.StructuredLogger) {
	lg.Debug(ctx, "Work started")
	time.Sleep(100 * time.Millisecond)
	lg.Warn(ctx, "Work is taking longer than expected (simulation)")
	lg.Debug(ctx, "Work finished")
}

func simulateError() error {
	return context.DeadlineExceeded
}
