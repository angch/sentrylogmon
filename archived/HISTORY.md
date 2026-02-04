# History

This file contains completed features and improvements for `sentrylogmon`, migrated from TODO.md.

## Completed Features

### Core
- **Structured Logging Support**: Implemented JSON log parsing (e.g., Zap, Logrus) with `JsonDetector` supporting `key:regex` pattern matching and metadata extraction.
- **Syslog Support**: Added `LogSource` for syslog (UDP/TCP) with severity and facility parsing.
- **Prometheus Metrics**: Exposed `/metrics` endpoint to track processed lines, detected issues, and Sentry errors using `github.com/prometheus/client_golang`.
- **Configuration Hot-Reload**: Implemented `fsnotify`-based config watching with self-exec for seamless reloads.
- **Multi-tenancy**: Enabled per-monitor Sentry DSN/Project configuration for routing logs to different projects.
- **Observability**: Added `/healthz` liveness probe and `last_activity_timestamp` metric.

### Improvements
- **Zig Pattern Matching**: Optimized string matching using `std.ascii.indexOfIgnoreCase` replacing the naive O(N*M) implementation.
- **Date Parsing**: Enhanced `extractTimestamp` to support Nginx Error and Access logs and handle timezones.
- **Rate Limiting**: Implemented configurable per-issue rate limiting.
- **Dynamic Clock Tick**: Used `sysconf` for accurate `CLK_TCK` detection in CPU usage calculations.
- **Lazy Load Process Command Lines in `sysstat` (Go)**: Optimized process stats collection by fetching command lines only for top processes, reducing I/O.
- **Optimize Buffering in `Monitor` (Go)**: Replaced string concatenation with `strings.Builder` to reduce allocations.
- **Optimize `DmesgDetector` Allocations (Go)**: Switched to `FindSubmatchIndex` and direct byte checks to reduce memory allocations by 65%.
- **Optimize `sysstat` System Refresh (Rust)**: Replaced `refresh_all()` with granular `refresh_memory()` and `refresh_processes()` to skip unnecessary sensor/network refreshes.

### Testing
- **Fuzz Testing**: Added fuzz tests for detector logic (`detectors/fuzz_test.go`).
- **End-to-End Tests**: Created containerized test suite using Docker Compose with `sentry-mock`.

### Rust Feature Parity
- **JsonDetector**: Implemented with `key:regex` matching.
- **IPC Mechanism**: Added Unix socket listener for status/update and self-restart.
- **Sysstat**: Added Disk Pressure monitoring and full command line argument retrieval.
- **Context Extraction**: Added `ContextExtractor` trait for returning metadata.

### Zig Feature Parity
- **IPC Mechanism**: Implemented Unix socket listener.
- **System Statistics**: Implemented collection of CPU, Memory, Load Average, and Top Processes.
- **JsonDetector**: Implemented basic JSON pattern matching.
- **Exclusion Patterns**: Added support for ignoring lines matching specific patterns.

## 2026-02 Cleanup

### Documentation Sync
- **Update Go Documentation**: Updated `README.md` to explicitly list Syslog (UDP/TCP) as a supported log source.
- **Update Rust Documentation**: Updated `rust/README.md` to note Syslog source status.
- **Update Zig Documentation**: Updated `zig/README.md` to note Syslog status and reflect implemented features (Journalctl, Config File, Batching).

### Rust Parity
- **Implement Syslog Source**: Implemented `SyslogSource` (UDP/TCP) in `sources/syslog.rs` and updated `main.rs`.
- **Update Status Output**: Improved `status` command to match Go's table format with TTY detection for JSON output.
- **Implement --format CLI Argument**: Added `--format` support to override detector defaults.
- **Implement Metrics Server**: Added `--metrics-port` and `/metrics`, `/healthz` endpoints.
- **Improve Status Output Alignment**: Implemented dynamic table formatting.
- **Implement pprof Support**: Investigated and implemented pprof-compatible endpoints.

### Zig Parity
- **Implement Syslog Source**: Implemented syslog receiver in `sources/syslog.zig`.
- **Update Status Output**: Updated `status` logic to match Go's table format.
- **Implement TTY Detection**: Added `isatty` check for JSON vs Table output.
- **Implement Metrics Server**: Added metrics server with `/metrics` and `/healthz`.
- **Improve Status Output Alignment**: Implemented tab-writer style alignment.
- **Implement pprof Support**: Investigated and implemented pprof-compatible endpoints.

### Performance Optimization
- **Refactor Timestamp Extraction for Performance**: Centralized timestamp parsing logic in `detectors/timestamps.go`, implemented `TimestampExtractor` interface in `DmesgDetector` and `NginxDetector`, and updated `Monitor` to use the interface to reduce memory allocations and CPU usage.

### General
- **Profile Memory Usage**: Profiled memory usage to identify bottlenecks.
