# Design Principles

## Project Motivation

### Problem Statement

Modern systems generate extensive logs from various sources (application logs, system journals, kernel messages), but monitoring all of them effectively can be challenging:

1. **Resource Constraints**: Traditional monitoring solutions can be CPU and memory intensive
2. **Fragmented Logging**: Logs are scattered across files, journalctl, dmesg, and custom outputs
3. **Alert Fatigue**: Need intelligent filtering to avoid noise while catching real issues
4. **Centralization**: Need to aggregate issues from multiple sources into a single monitoring platform

### Solution

`sentrylogmon` addresses these challenges by:

- Being **lightweight** - minimal resource footprint suitable for production deployment
- Supporting **multiple log sources** - unified interface for files, journals, and commands
- Providing **flexible pattern matching** - configurable detection of issues
- Integrating with **Sentry** - leveraging an established error tracking platform

## Design Choices and Rationale

### 1. Language: Go

**Choice**: Implemented in Go (Golang)

**Rationale**:
- **Low Resource Usage**: Go compiles to efficient native binaries with minimal memory overhead
- **Concurrency**: Built-in goroutines enable efficient handling of multiple log sources simultaneously
- **Single Binary**: Static linking produces self-contained executables - easy deployment
- **Cross-platform**: Excellent cross-compilation support for different Linux distributions and architectures
- **Mature Ecosystem**: Strong standard library and available Sentry SDK

### 2. Architecture: Simple and Focused

**Choice**: Single-purpose tool following Unix philosophy

**Rationale**:
- **Do One Thing Well**: Focus solely on log monitoring and Sentry forwarding
- **Composability**: Can be combined with other tools via standard Unix pipes and commands
- **Maintainability**: Smaller codebase is easier to understand, test, and maintain
- **Reliability**: Fewer features mean fewer failure points

### 3. Sentry Integration: CaptureMessage

**Choice**: Use Sentry's `CaptureMessage` API for forwarding log issues

**Rationale**:
- **Appropriate Semantic**: `CaptureMessage` is designed for log-like messages, not full exceptions
- **Metadata Support**: Can attach context like environment, release, tags
- **Rate Limiting**: Sentry's built-in rate limiting prevents overwhelming the service
- **Grouping**: Sentry intelligently groups similar messages
- **Alerting**: Leverage Sentry's existing alerting and notification infrastructure

### 4. Configuration: CLI Flags + Environment Variables

**Choice**: Support both command-line flags and environment variables for configuration

**Rationale**:
- **Flexibility**: CLI flags for ad-hoc usage, env vars for service deployment
- **12-Factor App**: Follows 12-factor app methodology for configuration
- **Container-Friendly**: Environment variables work well in Docker/Kubernetes
- **Security**: Sensitive data (DSN) can be kept out of command-line history via env vars

**Future Consideration**: Configuration files (YAML/TOML) could be added for complex multi-monitor setups without breaking existing CLI interface. (Implemented 2026-01-27)

### 5. Log Source Support

**Choice**: Support multiple log source types through a common interface

**Rationale**:
- **Files**: Most common log format, requires tail-like functionality
- **journalctl**: Native systemd journal access for modern Linux systems
- **dmesg**: Kernel logs important for system-level issues
- **Custom Commands**: Extensibility for any tool that outputs to stdout

**Implementation Note**: Each source type should implement a common `LogSource` interface with methods like `Read() (string, error)` and `Close() error`.

### 6. Pattern Matching: Regular Expressions

**Choice**: Use regex patterns for issue detection

**Rationale**:
- **Flexibility**: Regex provides powerful pattern matching capabilities
- **Standard**: Well-understood by developers and operators
- **Performance**: Go's regex engine (RE2) is safe and reasonably fast
- **Default Pattern**: Case-insensitive "error" catches most common issues

**Future Enhancement**: Could add support for structured log parsing (JSON) with field-based matching.
