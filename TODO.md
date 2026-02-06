# TODO - Work Items

This file tracks active work items.

## Completed

- [x] **Syslog Timestamp Optimization** (2026-02-06)
  - Implemented `ParseSyslogTimestamp` manual parser to replace regex.
  - Reduced allocations from 32B/op to 16B/op.
  - Improved performance by 3.6x (794ns -> 219ns).
  - Verified with new unit tests and benchmarks.
- [x] **JSON Log Support** (2026-02-05)
  - Implemented `ExtractTimestamp` in `JsonDetector`.
  - Optimized `JsonDetector` with thread-safe caching (mutex + byte comparison) to avoid double unmarshalling.
  - Added comprehensive tests for concurrency and cache consistency.
- [x] **JSON Severity Support** (2026-02-06)
  - Implemented extraction of `level`, `severity`, `log_level` from JSON logs.
  - Mapped string levels (e.g., "error", "warn") to `sentry.Level`.
  - Added `TestJsonSeverity` to verify mapping logic.
- [x] **DmesgDetector Optimization** (2026-02-07)
  - Implemented zero-allocation `parseFloatFromBytes` to avoid string conversion during timestamp parsing.
  - Optimized `areHeadersRelated` to accept `[]byte` and perform allocation-free comparisons.
  - Reduced allocations for `DmesgDetector.Detect` by 25% for error lines and 50% for context lines.
- [x] **ISO8601 Timestamp Optimization** (2026-02-08)
  - Implemented `ParseISO8601` manual parser to replace `time.Parse`.
  - Improved performance by 36% for standard RFC3339 (142ns -> 90ns) and 73% for space-separated format (270ns -> 73ns).
  - Maintained zero-allocation correctness (excluding return string).
  - Added comprehensive unit tests and benchmarks.
