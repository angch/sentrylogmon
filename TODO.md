# TODO - Wish List

This file tracks potential features and improvements for `sentrylogmon`.

## Features

- [x] **Structured Logging Support**
  - Parse JSON logs (e.g., from Zap, Logrus, Bunyan).
  - Match patterns against specific fields (e.g., `level == "error"`).
  - Extract metadata from log fields to populate Sentry tags/context.
  - *Implemented using `JsonDetector` with `key:regex` support.*

- [ ] **Syslog Support**
  - [x] Add a new `LogSource` for syslog (UDP/TCP listener).
  - [x] Parse syslog severity/facility.

- [x] **Prometheus Metrics**
  - Expose an HTTP endpoint (e.g., `/metrics`) with Prometheus metrics.
  - Track lines processed, issues detected, Sentry errors, etc.
  - *Implemented using `github.com/prometheus/client_golang`.*

- [ ] **Configuration Hot-Reload**
  - Watch the configuration file for changes.
  - Reload monitors without restarting the process (or graceful restart).

- [ ] **Multi-tenancy**
  - Support routing different log sources to different Sentry DSNs/Projects.

## Improvements

- [x] **Zig Pattern Matching Optimization**
  - The current `containsPattern` in Zig is a naive O(N*M) implementation.
  - Implement Boyer-Moore or use a proper regex library (if available/lightweight) or `std.mem.indexOf` optimizations.
  - *Implemented using `std.ascii.indexOfIgnoreCase`.*

- [x] **Better Date Parsing**
  - Support more timestamp formats in `extractTimestamp`.
  - Handle timezone conversions correctly.

- [x] **Rate Limiting**
  - Implement per-issue rate limiting to avoid flooding Sentry.
  - Configurable burst/rate.

## Testing

- [ ] **Fuzz Testing**
  - Fuzz the detector logic with random inputs to find edge cases.

- [ ] **End-to-End Tests**
  - Containerized tests using Docker Compose (Sentry mock + log generator + sentrylogmon).

## Feature Parity (Rust)

- [x] **Implement `JsonDetector`**
  - Support `key:regex` matching similar to Go.
- [x] **Implement IPC Mechanism**
  - Add Unix socket listener for `/status` and `/update`.
  - Implement self-restart logic.
- [ ] **Enhance `sysstat`**
  - [x] Add Disk Pressure collection (/proc/pressure/io).
  - [x] Retrieve full command line arguments for top processes (currently only process name).
- [x] **Implement Context Extraction**
  - Add `ContextExtractor` trait to detectors to allow returning metadata map.

## Feature Parity (Zig)

- [x] **Implement IPC Mechanism**
  - Add Unix socket listener for `/status` and `/update`.
- [x] **Implement System Statistics**
  - Collect CPU, Memory, Load Average, and Top Processes.
  - Attach system state to Sentry events.
- [x] **Implement `JsonDetector`**
  - Basic JSON parsing to match keys against patterns.
- [x] **Support Exclusion Patterns**
  - Implement logic to ignore lines matching an exclude pattern.
