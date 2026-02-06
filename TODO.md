# TODO - Work Items

This file tracks active work items.

## Completed

- [x] **JSON Log Support** (2026-02-05)
  - Implemented `ExtractTimestamp` in `JsonDetector`.
  - Optimized `JsonDetector` with thread-safe caching (mutex + byte comparison) to avoid double unmarshalling.
  - Added comprehensive tests for concurrency and cache consistency.
- [x] **JSON Severity Support** (2026-02-06)
  - Implemented extraction of `level`, `severity`, `log_level` from JSON logs.
  - Mapped string levels (e.g., "error", "warn") to `sentry.Level`.
  - Added `TestJsonSeverity` to verify mapping logic.
