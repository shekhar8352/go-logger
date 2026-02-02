# go-logger Development Plan

This document is the single source of truth for the go-logger roadmap. The plan is structured in **phases** with a gradual build-up: phases are ordered by dependency, and each phase is independently testable. Implement one phase at a time and add the tests listed for that phase before moving on.

---

## Table of contents

- [Phase 0: Foundation and resilience](#phase-0-foundation-and-resilience)
- [Phase 1: Config and reader consistency](#phase-1-config-and-reader-consistency)
- [Phase 2: Structured fields and child loggers](#phase-2-structured-fields-and-child-loggers)
- [Phase 3: Context and tracing](#phase-3-context-and-tracing)
- [Phase 4: Log format and timestamp](#phase-4-log-format-and-timestamp)
- [Phase 5: Rotation and retention](#phase-5-rotation-and-retention)
- [Phase 6: Output and async writer](#phase-6-output-and-async-writer)
- [Phase 7: Hooks and metrics](#phase-7-hooks-and-metrics)
- [Phase 8: Search and reader enhancements](#phase-8-search-and-reader-enhancements)
- [Phase 9: Sampling and final resilience](#phase-9-sampling-and-final-resilience)
- [Phase 10: Stack trace on error](#phase-10-stack-trace-on-error)
- [Appendix](#appendix)

---

## Phase 0: Foundation and resilience

### Description

Avoid panics in [logger/logging/rotate.go](logger/logging/rotate.go) when the log directory cannot be created or the log file cannot be opened. Return errors or fall back to stderr instead of panicking. Add a small **buffer/in-memory sink** (e.g. `BufferSink` or test helper) so tests can assert on log output without touching the filesystem.

### Agent prompt

```
Implement Phase 0 of the go-logger development plan.

1. In logger/logging/rotate.go: Replace the panic calls in rotateLogFile with error handling. When the log directory cannot be created or the log file cannot be opened, do not panic. Either return an error (and expose a way for NewLogger or the caller to detect/surface this) or fall back to writing to stderr so logging still works. Ensure the logger remains usable when rotation fails.

2. Add an in-memory sink type (e.g. BufferSink or a test helper) that implements the same write interface as the file (e.g. io.Writer). Allow the logger to be constructed or configured to use this sink so tests can assert on log output without touching the filesystem.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Rotation failure (e.g. unwritable dir or file open failure) no longer panics.
- Logger can be constructed with a buffer sink and writes go to that sink.
- Existing tests and benchmarks still pass.

### Tests to write

- `TestRotateLogFile_WhenDirFails_DoesNotPanic`
- `TestRotateLogFile_WhenFileOpenFails_DoesNotPanic`
- `TestLogger_WithBufferSink_WritesToBuffer`
- `TestBufferSink_ReturnsWrittenLines`

---

## Phase 1: Config and reader consistency

### Description

Make the reader/search APIs use the same log directory and file prefix as the writer. Today [logger/logging/reader.go](logger/logging/reader.go) uses `GetLogDirectory()` and hardcodes `"app"` in several places (e.g. `SearchLogs`, `ReadLogsByDate`). Add optional config (or explicit params) so callers can pass `LogsDir` and `FilePrefix` (e.g. `SearchLogsWithConfig(cfg, query, date, level)` or extend `SearchLogs` to accept an options struct). Default behavior should match `DefaultConfig()` (e.g. `./logs`, `app`).

### Agent prompt

```
Implement Phase 1 of the go-logger development plan.

Add reader APIs that accept config or an options struct containing LogsDir and FilePrefix. Refactor SearchLogs, ReadLogsByDate, and related helpers (e.g. GetAvailableLogFiles, GetDateRange usage) to use this config when provided. When no config is passed, keep the current default behavior (GetLogDirectory() and prefix "app") so existing callers are unchanged. Ensure the default behavior matches DefaultConfig() (./logs, app).

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Search/read use the provided config (LogsDir, FilePrefix) when given.
- Default behavior (no config) is unchanged.
- Existing examples still work.

### Tests to write

- `TestSearchLogs_WithConfig_UsesConfigDirAndPrefix`
- `TestReadLogsByDate_WithConfig_UsesConfigPath`
- `TestGetLogDirectory_RespectsEnv`
- `TestGetAvailableLogFiles_WithCustomPrefix`

---

## Phase 2: Structured fields and child loggers

### Description

Support key-value fields on log entries (e.g. `WithField("user_id", 123)`, `WithFields(map[string]interface{}{...})`) and **child loggers** that carry fixed fields (e.g. `logger.WithFields(...)` returns a logger that adds those fields to every entry). Extend [logger/logging/entry.go](logger/logging/entry.go) to store structured fields (e.g. a `Fields map[string]interface{}` or similar) and serialize them in JSON. Ensure [logger/logging/logger.go](logger/logging/logger.go) supports creating entries with these fields and that child loggers inherit base fields.

### Agent prompt

```
Implement Phase 2 of the go-logger development plan.

1. Add support for key-value fields on log entries: WithField(key, value) and/or WithFields(map[string]interface{}). These can be attached via context or an entry builder. Extend logger/logging/entry.go to store structured fields (e.g. Fields map[string]interface{}) and include them in JSON output.

2. Add StructuredLogger.WithFields(fields) that returns a child logger. Every log entry produced by the child must include these base fields. Ensure logger/logging/logger.go creates entries that include context-derived and extra fields, and that child loggers merge their fields into each entry.

3. Document usage (e.g. in package doc or README).

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Entries can carry key-value fields; they appear in JSON output.
- Child logger adds its fields to all log levels (Debug, Info, Warn, Error).
- Existing API remains backward compatible (calling without fields behaves as before).

### Tests to write

- `TestLogEntry_WithFields_SerializesToJSON`
- `TestLogger_WithFields_ChildAddsFieldsToEveryEntry`
- `TestCreateLogEntry_IncludesContextAndExtraFields`
- `TestParser_ParseLogLine_WithExtraFields`

---

## Phase 3: Context and tracing

### Description

Allow setting the log ID from outside (e.g. from an incoming request header) via `WithLogID(ctx, id)`, and support additional context keys such as `request_id`, `trace_id`, `user_id` in [logger/logging/context.go](logger/logging/context.go). Log entries should include these when present (e.g. in **LogEntry** or in a structured field in JSON).

### Agent prompt

```
Implement Phase 3 of the go-logger development plan.

1. In logger/logging/context.go: Add WithLogID(ctx, id) to set a specific log ID on the context (e.g. from an incoming request header). Keep AddLogID(ctx) to generate a new ID when none is set.

2. Add optional context helpers and keys for request_id, trace_id, user_id (e.g. WithRequestID, WithTraceID, WithUserID and corresponding getters). Include these values in LogEntry and in JSON output when present.

3. Ensure GetLogID and existing AddLogID behavior remain unchanged for callers that do not use the new APIs.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Log ID can be set from outside via WithLogID; it appears in log output.
- Optional context keys (request_id, trace_id, user_id) appear in log output when set.
- AddLogID still generates a new ID when none is present.

### Tests to write

- `TestWithLogID_OverridesLogID`
- `TestGetLogID_FromContext`
- `TestLogEntry_IncludesRequestIdWhenSet`
- `TestAddLogID_WhenIdNotSet_GeneratesNew`

---

## Phase 4: Log format and timestamp

### Description

Add configurable **log format** (JSON vs logfmt) and **timestamp format** (e.g. RFC3339 vs Unix milliseconds) in [logger/logging/config.go](logger/logging/config.go). Serialization in [logger/logging/logger.go](logger/logging/logger.go) should branch on format; timestamp formatting should use the configured layout.

### Agent prompt

```
Implement Phase 4 of the go-logger development plan.

1. In logger/logging/config.go: Add LogFormat (e.g. "json" or "logfmt") and TimestampFormat (or TimeFormat) to LoggerConfig. Use sensible defaults (e.g. json, RFC3339) so existing behavior is unchanged.

2. In logger/logging/logger.go: When writing a log entry, branch on LogFormat to serialize as JSON or as logfmt. Use the configured TimestampFormat when formatting the entry timestamp.

3. Ensure the existing parser still correctly parses JSON lines; logfmt parsing is optional for this phase.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Both JSON and logfmt produce valid, readable output.
- Timestamp format option changes the timestamp in each entry.
- Parser (ParseLogLine) still works for JSON output.

### Tests to write

- `TestLogger_LogFormat_JSON`
- `TestLogger_LogFormat_Logfmt`
- `TestLogger_TimestampFormat_AppliesToEntry`
- `TestParseLogLine_StillWorksForJSON`

---

## Phase 5: Rotation and retention

### Description

Add **size-based rotation** (rotate when file exceeds N bytes), **retention/cleanup** (delete or archive files older than N days or keep last N files), and optional **compression** (gzip rotated/closed files). Extend [logger/logging/rotate.go](logger/logging/rotate.go) and config (e.g. `MaxFileSize`, `RetentionDays`, `CompressRotated`).

### Agent prompt

```
Implement Phase 5 of the go-logger development plan.

1. Add size-based rotation: when the current log file exceeds MaxFileSize (config), rotate to a new file. Integrate this with existing daily rotation so both policies can apply (e.g. rotate on day change or when size is exceeded).

2. Add retention policy: optionally delete (or archive) log files older than RetentionDays, or keep only the last N files. Run cleanup as part of rotation or on a schedule that makes sense.

3. Add optional compression: when CompressRotated is true, gzip rotated/closed log files (e.g. app-2026-01-22.log.gz). Document behavior and config fields.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Rotation happens when the current file exceeds the configured size limit.
- Old files are removed or compressed according to config.
- No data loss under normal operation (all lines are written before rotation).

### Tests to write

- `TestRotate_WhenSizeExceeded_RotatesFile`
- `TestRetention_DeletesFilesOlderThanN`
- `TestCompressRotated_CreatesGzipFile`
- `TestRotation_DoesNotLoseLines`

---

## Phase 6: Output and async writer

### Description

Add **async/buffered writer** (channel or buffer, flush on interval or on close), **stderr by level** (e.g. ERROR/WARN to stderr when `IncludeConsole` is true), and **multiple sinks** (e.g. file + stdout, or file + custom `io.Writer`). Config might include `UseAsyncWriter`, `FlushInterval`, `StderrLevels`, and `Sinks []io.Writer`.

### Agent prompt

```
Implement Phase 6 of the go-logger development plan.

1. Add an optional async write path: when UseAsyncWriter is true, buffer log entries and write them in a background goroutine. Support configurable buffer size and FlushInterval. Ensure Close() flushes the buffer so no entries are lost on shutdown.

2. When IncludeConsole is true, route output by level: e.g. ERROR (and optionally WARN) to stderr, others to stdout. Make this configurable (e.g. StderrLevels) so callers can choose which levels go to stderr.

3. Support multiple sinks: in addition to the main log file and console, allow extra io.Writer sinks (e.g. Sinks []io.Writer) that receive the same log output.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Async logger does not block callers; writes happen in the background.
- Flush on Close delivers all buffered entries.
- Stderr receives only the configured levels; stdout receives the rest.
- Multiple sinks all receive the same entries.

### Tests to write

- `TestAsyncWriter_FlushesOnClose`
- `TestAsyncWriter_DoesNotBlock`
- `TestLogger_StderrLevels_ErrorsToStderr`
- `TestLogger_MultipleSinks_AllReceive`

---

## Phase 7: Hooks and metrics

### Description

Add **hooks** (callbacks invoked per log entry or per level) and optional **metrics** (e.g. counters per level). Config could include `Hooks []func(LogEntry)` or `OnError func(LogEntry)`; optional metrics interface or simple counters.

### Agent prompt

```
Implement Phase 7 of the go-logger development plan.

1. Add hook registration to LoggerConfig (e.g. Hooks []func(LogEntry) or OnError func(LogEntry)). In writeLog, after building the entry and before or after writing, invoke each hook with the log entry. Document the hook contract (e.g. do not block; avoid panics).

2. Optionally add simple metrics: e.g. counters per level (Debug, Info, Warn, Error) that can be read for monitoring. If metrics are in scope, add a minimal interface or struct to expose these counts.

3. Ensure level filtering (MinLevel) is applied before hooks so hooks are only called for entries that would be logged. Ensure slow hooks do not deadlock the logger (e.g. run hooks in a separate goroutine or with a timeout, and document behavior).

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Hooks are called with the correct LogEntry for each written log.
- Level filtering (MinLevel) is respected; hooks are not called for filtered-out entries.
- No deadlocks when hooks are slow (e.g. test with a timeout).

### Tests to write

- `TestHook_InvokedWithEntry`
- `TestHook_RespectsMinLevel`
- `TestHook_SlowHook_DoesNotBlockLogger` (or timeout-based test)

---

## Phase 8: Search and reader enhancements

### Description

Add **pagination** (offset/limit or cursor) to search and read APIs; **time-range filter** (filter by time of day or timestamp range); **tail/stream** API (follow current log file); optional **regex search** for message/path. Extend [logger/logging/reader.go](logger/logging/reader.go) and related types.

### Agent prompt

```
Implement Phase 8 of the go-logger development plan.

1. Add pagination to search and read APIs: accept offset and limit (or a cursor) and return only the requested slice of results. Keep existing APIs unchanged; add new functions or optional parameters (e.g. SearchLogsWithOptions with Pagination).

2. Add a time-range filter: allow filtering log entries by timestamp range (e.g. start time and end time) in addition to date. Apply this in ReadAndFilterLogs or a new helper.

3. Add a Tail or Follow API that reads new lines from the current log file as they are written (e.g. similar to tail -f). This can return a channel or callback for new lines.

4. Add optional regex search: allow the search query to be interpreted as a regex for matching message or path, when enabled (e.g. via options). Keep substring search as default.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- Pagination returns the correct slice (offset/limit) of results.
- Time-range filter excludes entries outside the range.
- Tail/Follow returns new lines as they are written to the current log file.
- Regex search matches as expected when enabled.

### Tests to write

- `TestSearchLogs_WithPagination_ReturnsCorrectSlice`
- `TestReadLogs_WithTimeRange_Filters`
- `TestTail_ReturnsNewLines`
- `TestSearchLogs_WithRegex_Matches`

---

## Phase 9: Sampling and final resilience

### Description

Add optional **sampling** for high-volume logs (e.g. log 1-in-N at DEBUG). Document **graceful degradation** (already partially in Phase 0) and any fallback behavior.

### Agent prompt

```
Implement Phase 9 of the go-logger development plan.

1. Add sampling configuration to LoggerConfig (e.g. SampleRate for a given level or SampleEveryN). When sampling is enabled for a level, log only a subset of entries (e.g. 1-in-N). Apply sampling only when explicitly enabled; when disabled, behavior is unchanged.

2. Document the sampling behavior (e.g. which levels support it, how the rate is applied) and document graceful degradation (fallback to stderr or buffer when rotation fails, as in Phase 0).

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- When sampling is enabled, the approximate rate of logged entries matches the config (e.g. roughly 1-in-N).
- When sampling is disabled, all entries are logged as before.

### Tests to write

- `TestLogger_WithSampling_ReducesVolume`
- `TestLogger_WithoutSampling_LogsAll`

---

## Phase 10: Stack trace on error

### Description

Optionally attach a **stack trace** to ERROR (and optionally WARN) entries. Config flag e.g. `IncludeStackTraceOnError`; use `runtime/debug.Stack()` or similar.

### Agent prompt

```
Implement Phase 10 of the go-logger development plan.

1. Add a config option (e.g. IncludeStackTraceOnError bool) to LoggerConfig. When true and the log level is ERROR (and optionally WARN), attach a stack trace to the log entry (e.g. via runtime/debug.Stack() or runtime.Callers). Add a field to LogEntry for the stack trace (e.g. StackTrace string) and include it in JSON output.

2. When the option is false, do not add any stack trace field; existing behavior is unchanged.

Preserve backward compatibility and add the tests listed in this phase.
```

### Testing criteria

- When enabled, ERROR entries contain a non-empty stack trace in the output.
- When disabled, no stack trace field is added; behavior is unchanged.

### Tests to write

- `TestError_WithStackTrace_IncludesStack`
- `TestError_WithoutStackTrace_NoStackField`

---

## Appendix

### Test layout

- All new tests live in [logger/logging/](logger/logging/) (e.g. `logger_test.go`, `rotate_test.go`, `reader_test.go`, `context_test.go` as needed).
- Prefer table-driven tests where multiple cases are covered.
- Keep benchmarks in `logger_test.go` or a dedicated `_bench_test.go` file.

### Backward compatibility

- Each phase must keep existing public API behavior unless the plan explicitly breaks it.
- New APIs can be added alongside old ones (e.g. `SearchLogsWithConfig` in addition to `SearchLogs`).
- Default config values must remain so that existing callers see no change in behavior.
