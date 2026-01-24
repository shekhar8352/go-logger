package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shekhar8352/go-logger/logger/logging"
)

// LoadConfigFromEnv simulates loading config from environment variables
func LoadConfigFromEnv() logging.LoggerConfig {
	cfg := logging.DefaultConfig()

	if dir := os.Getenv("APP_LOG_DIR"); dir != "" {
		cfg.LogsDir = dir
	}
	if prefix := os.Getenv("APP_LOG_PREFIX"); prefix != "" {
		cfg.FilePrefix = prefix
	}
	if level := os.Getenv("APP_LOG_LEVEL"); level != "" {
		cfg.MinLevel = level
	}
	// Note: In a real app we would parse boolean properly
	return cfg
}

func main() {
	// 1. Simulate setting environment variables
	os.Setenv("APP_LOG_LEVEL", "ERROR")
	os.Setenv("APP_LOG_PREFIX", "service-x")

	fmt.Println("Initializing logger from environment variables...")
	cfg := LoadConfigFromEnv()
	lg := logging.NewLogger(cfg)
	defer lg.Close()

	// Override the global default logger if you want 'logging.Info()' to use this config
	logging.DefaultLogger = lg

	ctx := context.Background()

	// 2. These Info logs should NOT appear because MinLevel is ERROR
	logging.Info(ctx, "This info message will be hidden")
	lg.Info(ctx, "This info message will also be hidden")

	// 3. This Error log SHOULD appear
	lg.Error(ctx, "This critical error will be visible")

	// 4. Runtime Level Change
	fmt.Println("Changing log level to DEBUG at runtime...")
	lg.SetLevel(logging.LevelDebug)

	lg.Debug(ctx, "Now debugging is enabled and visible")
	lg.Info(ctx, "Info messages are now back")
}
