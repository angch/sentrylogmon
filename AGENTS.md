# AGENTS.md - Guide for LLM Agents

This document is intended for LLM agents working on future enhancements to the sentrylogmon project. It explains the motivation, design choices, and architectural decisions made during development.

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

**Future Consideration**: Configuration files (YAML/TOML) could be added for complex multi-monitor setups without breaking existing CLI interface.

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

## Code Structure Guidelines

### Recommended Project Structure

```
sentrylogmon/
├── main.go                 # Entry point, CLI parsing, main loop
├── config/
│   └── config.go          # Configuration struct and parsing
├── sources/
│   ├── source.go          # LogSource interface definition
│   ├── file.go            # File-based log source
│   ├── journalctl.go      # journalctl log source
│   ├── dmesg.go           # dmesg log source
│   └── command.go         # Custom command log source
├── detector/
│   └── detector.go        # Pattern matching and detection logic
├── forwarder/
│   └── sentry.go          # Sentry integration and message forwarding
├── go.mod
├── go.sum
├── README.md
├── AGENTS.md
└── LICENSE
```

### Key Abstractions

#### LogSource Interface

```go
type LogSource interface {
    // Read returns the next log line or an error
    Read() (string, error)
    
    // Close releases resources associated with the log source
    Close() error
}
```

#### Detector

```go
type Detector struct {
    pattern *regexp.Regexp
}

func (d *Detector) IsIssue(line string) bool {
    return d.pattern.MatchString(line)
}
```

#### Forwarder

```go
type SentryForwarder struct {
    hub *sentry.Hub
}

func (f *SentryForwarder) Forward(message string, context map[string]interface{}) error {
    // Send to Sentry using CaptureMessage
}
```

## Technology Decisions

### Core Dependencies

1. **github.com/getsentry/sentry-go**: Official Sentry Go SDK
   - Well-maintained and officially supported
   - Handles connection pooling, retries, and rate limiting
   - Supports all Sentry features (breadcrumbs, contexts, etc.)

2. **Standard Library**: Prefer stdlib over third-party where possible
   - `regexp` for pattern matching
   - `flag` for CLI parsing (or `github.com/spf13/cobra` for more complex CLI)
   - `bufio` for efficient line reading
   - `os/exec` for running commands

### Optional Dependencies (Future)

- **github.com/spf13/cobra**: For rich CLI with subcommands
- **github.com/spf13/viper**: For configuration file support
- **gopkg.in/yaml.v3**: For YAML config parsing
- **github.com/fsnotify/fsnotify**: For file watching instead of polling

## Future Enhancement Guidelines

### Planned Features

1. **Configuration File Support**
   - YAML/TOML format for defining multiple monitors
   - Hot-reload capability to update config without restart
   - Validate config before applying

2. **Metrics and Health Monitoring**
   - Expose Prometheus metrics (lines processed, issues detected, etc.)
   - Health check endpoint for monitoring the monitor itself
   - Self-monitoring: detect if log sources stop producing output

3. **Buffering and Batching**
   - Buffer messages during Sentry outages
   - Batch similar messages to reduce API calls
   - Persistent queue for guaranteed delivery

4. **Advanced Filtering**
   - Whitelist patterns (exclude matches from reporting)
   - Rate limiting per pattern to avoid spam
   - Time-based filtering (only monitor during business hours)
   - Severity levels

5. **Structured Log Support**
   - Parse JSON logs and match on fields
   - Extract metadata from structured logs for Sentry context
   - Support for common formats (Logrus, Zap, etc.)

6. **Multi-tenancy**
   - Support multiple Sentry projects/DSNs
   - Route different log sources to different Sentry projects

### Non-Goals

The following are explicitly **not** goals for this project:

1. **Log Aggregation Storage**: Use Elasticsearch, Loki, or similar for this
2. **Complex Analytics**: Use dedicated log analysis tools
3. **Log Transformation**: Keep the tool focused on monitoring and forwarding
4. **GUI**: Remain a CLI tool; use Sentry's web interface for visualization
5. **Plugin System**: Keep the codebase simple; fork if extensive customization needed

## Testing Strategy

### Unit Tests

- Test each LogSource implementation independently
- Mock Sentry client for Forwarder tests
- Test pattern matching edge cases in Detector
- Aim for >80% code coverage

### Integration Tests

- End-to-end tests with real log files
- Test error handling and recovery
- Verify Sentry integration with test DSN

### Performance Tests

- Benchmark memory usage with large log files
- Test CPU usage under sustained load
- Verify resource usage stays within "lightweight" criteria (<50MB RAM, <5% CPU)

## Common Pitfalls for Future Development

### 1. File Monitoring

**Issue**: Using polling instead of inotify/fsnotify can be inefficient

**Solution**: Consider `github.com/fsnotify/fsnotify` for production deployments, but keep polling as fallback for compatibility

### 2. Memory Leaks with Long-Running Processes

**Issue**: Go applications can leak memory with improper goroutine management

**Solution**: 
- Always defer `Close()` calls
- Use context for cancellation
- Profile with pprof regularly: `import _ "net/http/pprof"`

### 3. Regex Performance

**Issue**: Complex regexes can be slow on high-volume logs

**Solution**:
- Pre-compile regexes once
- Use simple patterns when possible
- Consider Boyer-Moore or other algorithms for literal string matching

### 4. Sentry Rate Limiting

**Issue**: Sending too many messages can hit Sentry rate limits

**Solution**:
- Implement client-side rate limiting
- Batch similar messages
- Use sampling for high-frequency issues

### 5. Timezone Handling

**Issue**: Log timestamps may be in different timezones

**Solution**:
- Always normalize to UTC
- Preserve original timezone in metadata
- Use Go's `time.Time` type throughout

## Contribution Guidelines for Agents

When enhancing this project:

1. **Maintain Simplicity**: Don't add features that significantly increase complexity
2. **Performance First**: Profile before and after changes; maintain lightweight footprint
3. **Backward Compatibility**: Preserve existing CLI flags and behavior
4. **Document Decisions**: Update this AGENTS.md with rationale for significant changes
5. **Test Coverage**: Add tests for new functionality
6. **Error Handling**: Follow Go conventions; return errors, don't panic
7. **Logging**: Use structured logging; respect --verbose flag

## Resources

- **Go Best Practices**: https://go.dev/doc/effective_go
- **Sentry Go SDK Docs**: https://docs.sentry.io/platforms/go/
- **12-Factor App**: https://12factor.net/
- **Unix Philosophy**: https://en.wikipedia.org/wiki/Unix_philosophy

## Changelog of Major Decisions

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-27 | Initial design using Go + Sentry | Best balance of performance and integration capabilities |
| 2026-01-27 | CLI flags + env vars for config | Simplicity and container-friendliness |
| 2026-01-27 | Support for files, journalctl, dmesg | Cover 90% of common use cases |

---

**Last Updated**: 2026-01-27
**Maintained By**: Project contributors and LLM agents