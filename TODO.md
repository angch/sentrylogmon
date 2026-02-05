# TODO - Work Items

This file tracks active work items.

## Completed

- [x] **JSON Log Support** (2026-02-05)
  - Implemented `ExtractTimestamp` in `JsonDetector`.
  - Optimized `JsonDetector` with thread-safe caching (mutex + byte comparison) to avoid double unmarshalling.
  - Added comprehensive tests for concurrency and cache consistency.
