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
