# Logger Examples

This directory contains examples demonstrating various features of `go-logger`.

## 1. Basic Usage (`examples/basic`)

Shows the fundamental usage of the logger:

- Initializing the logger.
- Creating a `LogID` and attaching it to a context.
- Logging at different levels (Info, Debug, Error).
- Demonstrates how the logger handles file writing and console output.

**Run:**

```bash
go run logger/examples/basic/main.go
```

## 2. Search & Read Logs (`examples/search`)

Demonstrates the powerful log retrieval capabilities:

- Generating logs with specific keywords and IDs.
- Using `logging.SearchLogs` to find logs by keyword within a specific date range or level.
- Viewing the structured `LogEntry` results.

**Run:**

```bash
go run logger/examples/search/main.go
```

## 3. HTTP Middleware (`examples/http-middleware`)

A real-world example of integrating the logger into an HTTP server:

- Custom middleware `LoggingMiddleware` that injects a unique `LogID` into every request's context.
- Logs the start of the request, the method, and the path.
- Logs the completion of the request along with the execution duration.
- Shows how handlers can retrieve the logger from the context.

**Run:**

```bash
go run logger/examples/http-middleware/main.go
```

_Then visit `http://localhost:8080/hello` in your browser or curl._

## 4. Advanced Configuration (`examples/advanced-config`)

Shows how to manage configuration dynamically:

- Loading configuration from environment variables (simulated).
- Changing log levels at runtime (e.g., switching from ERROR to DEBUG without restarting).
- Overriding the global `DefaultLogger` so all package-level calls use the new config.

**Run:**

```bash
go run logger/examples/advanced-config/main.go
```
