package main

import (
	"context"
	"fmt"
	"time"

	"github.com/shekhar8352/go-logger/logger/logging"
)

func main() {
	// 1. Setup Logger to generate some data first
	lg := logging.NewLogger(logging.DefaultConfig())
	defer lg.Close()

	ctx := logging.AddLogID(context.Background())
	lg.Info(ctx, "Generating some searchable data point A")
	lg.Error(ctx, "This is a test error for search functionality")
	lg.Info(ctx, "Another data point B with unique id: %s", "UID-12345")

	// Allow file system sync
	time.Sleep(100 * time.Millisecond)

	fmt.Println("--- Searching for 'error' ---")
	// 2. Search for logs containing "error"
	// SearchLogs params: query, specificDate (YYYY-MM-DD), level filter
	logs, count, err := logging.SearchLogs("error", "", "")
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
	} else {
		fmt.Printf("Found %d results for 'error':\n", count)
		printLogs(logs)
	}

	fmt.Println("\n--- Searching for 'UID-12345' ---")
	// 3. Search for specific data
	logs, count, err = logging.SearchLogs("UID-12345", "", "INFO")
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
	} else {
		fmt.Printf("Found %d results for 'UID-12345':\n", count)
		printLogs(logs)
	}
}

func printLogs(logs []logging.LogEntry) {
	for _, l := range logs {
		fmt.Printf("[%s] [%s] %s (LogID: %s)\n", l.Timestamp, l.Level, l.Message, l.LogID)
	}
}
