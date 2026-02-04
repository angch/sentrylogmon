# TODO - Work Items

This file tracks active work items.

## Performance Optimization

- [x] **Refactor Timestamp Extraction for Performance**
  - Create `detectors/timestamps.go` to centralize timestamp parsing logic.
  - Implement `TimestampExtractor` interface in `DmesgDetector` and `NginxDetector`.
  - Update `Monitor` to use the interface, avoiding regex guessing where possible.
  - Goal: Reduce memory allocations and CPU usage for timestamp extraction.
