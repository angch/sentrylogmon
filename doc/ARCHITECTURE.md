# Architecture & Code Structure

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
├── detectors/
│   ├── detector.go        # Pattern matching interface
│   └── ...                # Specific detector implementations
├── monitor/
│   └── monitor.go         # Orchestration and Sentry forwarding
├── sysstat/
│   └── sysstat.go         # System state collection
├── testdata/
│   └── ...                # Data-driven test files
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
    // Stream returns a reader that streams the log output.
    // It should handle starting the underlying process if necessary.
    Stream() (io.Reader, error)

    // Close stops the log source and releases resources.
    Close() error

    // Name returns the name of the source (e.g. for logging).
    Name() string
}
```

#### Detector

```go
type Detector interface {
    Detect(line string) bool
}
```

#### Monitor (Forwarder)

The `Monitor` struct in `monitor/monitor.go` handles the core logic:
1. Reads from `LogSource`
2. Checks lines against `Detector`
3. If issue detected, collects `SystemState`
4. Buffers and sends to Sentry

#### System State Collection (`sysstat`)

The `sysstat` package collects system metrics (Load, Memory, Top Processes) to provide context when an error occurs. This helps in diagnosing if the error was caused by resource exhaustion.

## Technology Decisions

### Core Dependencies

1. **github.com/getsentry/sentry-go**: Official Sentry Go SDK
   - Well-maintained and officially supported
   - Handles connection pooling, retries, and rate limiting
   - Supports all Sentry features (breadcrumbs, contexts, etc.)

2. **System Statistics**:
   - `github.com/shirou/gopsutil/v3`: For portable system stats (load, memory)
   - `github.com/prometheus/procfs`: For efficient access to /proc filesystem

3. **Standard Library**: Prefer stdlib over third-party where possible
   - `regexp` for pattern matching
   - `flag` for CLI parsing (or `github.com/spf13/cobra` for more complex CLI)
   - `bufio` for efficient line reading
   - `os/exec` for running commands

### Optional Dependencies (Future)

- **github.com/spf13/cobra**: For rich CLI with subcommands
- **github.com/spf13/viper**: For configuration file support
- **gopkg.in/yaml.v3**: For YAML config parsing
- **github.com/fsnotify/fsnotify**: For file watching instead of polling
