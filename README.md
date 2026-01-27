# sentrylogmon

A lightweight, resource-efficient log monitoring tool that watches various log sources for issues and forwards them to Sentry for centralized error tracking and alerting.

**Available in Go, Rust, and Zig implementations** - choose the one that best fits your environment!

## Overview

`sentrylogmon` is designed to be a minimal-overhead monitoring solution that can:
- Monitor log files in real-time
- Parse output from `journalctl` (systemd journal)
- Watch `dmesg` kernel logs
- Execute custom commands and monitor their output
- Detect issues based on configurable patterns (e.g., lines containing "error", "fatal", "panic")
- Send detected issues to Sentry using `CaptureMessage` for centralized tracking

The tool is optimized for low CPU and memory usage, making it suitable for deployment on production systems without impacting performance.

**Three implementations are available:**
- **Go version**: Full-featured reference implementation with regex support and official Sentry SDK (~7MB binary)
- **Rust version**: Async implementation with smaller binary (~3.6MB, 50% smaller than Go)
- **Zig version**: Ultra-lightweight port optimized for minimal resource usage (~200KB, 95% smaller than Go!)

See [rust/README.md](rust/README.md) for Rust-specific documentation and [zig/README.md](zig/README.md) for Zig-specific documentation.

## Features

- **Multiple Log Sources**: Support for files, journalctl, dmesg, and custom command outputs
- **Pattern-based Detection**: Configurable regex patterns to identify issues
- **Sentry Integration**: Direct integration with Sentry for error tracking and alerting
- **System Status Context**: Automatically captures and attaches system state (CPU load, memory usage, top processes) to Sentry events
- **Efficient File Watching**: Uses `fsnotify` for native file system notifications
- **Lightweight**: Minimal CPU and memory footprint
- **Flexible Configuration**: Command-line flags and environment variables
- **Real-time Monitoring**: Continuous monitoring with configurable check intervals
- **Three Implementations**: Choose between Go (reference), Rust (smaller binary), or Zig (smallest binary)

## Rust Port

A Rust implementation is available in the `rust/` directory. It provides the same functionality as the Go version but with:
- **Smaller binary size**: ~3.6MB vs ~7.2MB (50% smaller than Go)
- **Async I/O**: Built on Tokio for efficient non-blocking operations
- **Same features**: All core functionality matches the Go version

See [rust/README.md](rust/README.md) for Rust-specific documentation.

## Zig Port

A Zig implementation is available in the `zig/` directory. It provides core functionality with:
- **Smallest binary size**: ~200-300KB (95% smaller than Go, 90% smaller than Rust)
- **Minimal memory footprint**: ~2-3MB RSS vs ~5-8MB for Go
- **Zero external dependencies**: Only uses Zig standard library
- **Fast startup**: ~5-10ms vs ~50-100ms for Go
- **Simple pattern matching**: Case-insensitive substring search (simpler than full regex)

See [zig/README.md](zig/README.md) for Zig-specific documentation.

## Installation

### Prerequisites

**For Go version:**
- Go 1.19 or later

**For Rust version:**
- Rust 1.70 or later
- Cargo (comes with Rust)

**For Zig version:**
- Zig 0.13.0 or later

**Check prerequisites:**
```bash
make check-prereqs
```

**Install prerequisites (if missing):**
```bash
make install-prereqs
```

### Building from Source

**Build Go version:**
```bash
git clone https://github.com/angch/sentrylogmon.git
cd sentrylogmon
make build-go
```

**Build Rust version:**
```bash
cd sentrylogmon
make build-rust
```

**Build Zig version:**
```bash
cd sentrylogmon
make build-zig
```

**Build all three:**
```bash
make build-all
```

**Compare binary sizes:**
```bash
make compare-size
```

### Installing via go install

```bash
go install github.com/angch/sentrylogmon@latest
```


Specify the Sentry DSN via command line or environment variable:

```bash
# Command line
sentrylogmon --dsn="https://your-sentry-dsn@sentry.io/project"

# Environment variable
export SENTRY_DSN="https://your-sentry-dsn@sentry.io/project"
sentrylogmon --file=/var/log/app.log
```

#### Log Sources

**Monitor a log file:**
```bash
sentrylogmon --dsn="..." --file=/var/log/syslog
```

**Monitor multiple log files (glob pattern):**
```bash
sentrylogmon --dsn="..." --file="/var/log/*.log"
```

**Monitor journalctl output:**
```bash
sentrylogmon --dsn="..." --journalctl="--unit=myapp.service -f"
```

**Monitor dmesg:**
```bash
sentrylogmon --dsn="..." --dmesg
```

**Monitor custom command output:**
```bash
sentrylogmon --dsn="..." --command="tail -f /var/log/custom.log"
```

#### Detection Patterns

Customize the patterns used to detect issues:

```bash
# Default: matches lines containing "error" (case-insensitive)
sentrylogmon --dsn="..." --file=/var/log/app.log --pattern="(?i)error"

# Multiple patterns
sentrylogmon --dsn="..." --file=/var/log/app.log --pattern="(?i)(error|fatal|panic)"
```

#### Other Options

- `--interval`: Check interval in seconds (default: 10)
- `--environment`: Sentry environment tag (e.g., "production", "staging")
- `--release`: Sentry release identifier
- `--verbose`: Enable verbose logging
- `--oneshot`: Run once and exit when input stream ends (useful for batch processing or benchmarking)

### Configuration File

You can also use a configuration file to manage multiple monitors and settings. This is the recommended way for complex setups.

Create `sentrylogmon.yaml`:

```yaml
# sentrylogmon.yaml
sentry:
  dsn: https://your-dsn@sentry.io/project
  environment: production
  release: v1.2.3

monitors:
  - name: nginx-errors
    type: file
    path: /var/log/nginx/error.log
    format: nginx

  - name: all-logs
    type: file
    path: /var/log/*.log

  - name: app-errors
    type: file
    path: /var/log/app.log
    pattern: "(?i)(error|critical)"

  - name: app-journal
    type: journalctl
    args: "--unit=myapp.service -f"
    pattern: "(?i)(error|fatal|panic)"
```

Run with configuration file:
```bash
sentrylogmon --config=sentrylogmon.yaml
```

**Note:** If you provide Sentry configuration (DSN, environment, release) via flags or environment variables, they will be used as fallbacks if missing from the configuration file.

### Instance Management (IPC)

The Go version of `sentrylogmon` supports managing running instances via a secure IPC mechanism (Unix Domain Sockets). This allows you to list running instances and instruct them to restart (e.g., to pick up a new binary or configuration).

**List running instances:**
```bash
sentrylogmon --status
```
Output:
```json
[
  {
    "pid": 1234,
    "start_time": "2023-10-27T10:00:00Z",
    "version": "v1.0.0",
    "config": { ... }
  }
]
```

**Restart all running instances:**
```bash
sentrylogmon --update
```
This command sends a signal to all discovered instances to gracefully shut down their monitors and re-execute the binary in-place (preserving the PID). This is useful for upgrades or configuration reloading without stopping the service manually.

### Example Configurations

**Production web server monitoring:**
```bash
sentrylogmon \
  --dsn="https://your-dsn@sentry.io/project" \
  --file=/var/log/nginx/error.log \
  --pattern="(?i)(error|critical|alert|emerg)" \
  --environment=production \
  --release=v1.2.3
```

**System journal monitoring:**
```bash
sentrylogmon \
  --dsn="https://your-dsn@sentry.io/project" \
  --journalctl="--priority=err -f" \
  --environment=production
```

**Kernel log monitoring:**
```bash
sentrylogmon \
  --dsn="https://your-dsn@sentry.io/project" \
  --dmesg \
  --pattern="(?i)(error|fail|panic|oops)"
```

## Running as a Service

### Automated Installation (Recommended)

An installation script is provided to automatically build the binary, install it to `/usr/local/bin/`, create a default configuration, and set up the systemd service.

```bash
sudo ./install_service.sh
```

Follow the on-screen instructions to edit the configuration file (`/etc/sentrylogmon.yaml`) with your Sentry DSN before starting the service.

### Manual systemd Service Setup

If you prefer to configure it manually:

1. Copy the binary to `/usr/local/bin/`.
2. Create `/etc/systemd/system/sentrylogmon.service` (or use the provided `sentrylogmon.service` file):

```ini
[Unit]
Description=Sentry Log Monitor
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sentrylogmon --config=/etc/sentrylogmon.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable sentrylogmon
sudo systemctl start sentrylogmon
sudo systemctl status sentrylogmon
```

## Testing Utility: loggen

A utility tool `loggen` is included to generate dummy logs for testing and benchmarking purposes.

### Building loggen

```bash
go build -o loggen cmd/loggen/main.go
```

### Usage

Generate 100MB of Nginx-formatted logs with errors:

```bash
./loggen --size=100MB --format=nginx --error-rate=5.0 > test.log
```

**Options:**
- `--size`: Total size to generate (e.g., "100MB", "1GB").
- `--format`: Log format ("nginx", "dmesg").
- `--error-rate`: Percentage of error logs (0-100).

## Development

### Building

**Go version:**
```bash
make build-go
# or
go build -o sentrylogmon
```

**Rust version:**
```bash
make build-rust
# or
cd rust && cargo build --release
```

**Both versions:**
```bash
make build-all
```

### Testing

**Go tests:**
```bash
make test-go
# or
go test ./...
```

**Rust tests:**
```bash
make test-rust
# or
cd rust && cargo test
```

**All tests:**
```bash
make test-all
```

### Linting

**Go:**
```bash
golangci-lint run
```

**Rust:**
```bash
cd rust && cargo clippy
```

### Binary Size Comparison

```bash
make build-all
# Shows sizes of both binaries
# Go binary:   ~7.2MB
# Rust binary: ~3.6MB (50% smaller)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- Issues: https://github.com/angch/sentrylogmon/issues
- Sentry Documentation: https://docs.sentry.io/

## Acknowledgments

- Built with the [Sentry Go SDK](https://github.com/getsentry/sentry-go)
- Inspired by the need for lightweight, focused monitoring solutions
