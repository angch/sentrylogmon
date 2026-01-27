# sentrylogmon

A lightweight, resource-efficient log monitoring tool that watches various log sources for issues and forwards them to Sentry for centralized error tracking and alerting.

## Overview

`sentrylogmon` is designed to be a minimal-overhead monitoring solution that can:
- Monitor log files in real-time
- Parse output from `journalctl` (systemd journal)
- Watch `dmesg` kernel logs
- Execute custom commands and monitor their output
- Detect issues based on configurable patterns (e.g., lines containing "error", "fatal", "panic")
- Send detected issues to Sentry using `CaptureMessage` for centralized tracking

The tool is optimized for low CPU and memory usage, making it suitable for deployment on production systems without impacting performance.

**Two implementations are available:**
- **Go version**: Full-featured reference implementation with regex support and official Sentry SDK (~10MB binary)
- **Zig version**: Ultra-lightweight port optimized for minimal resource usage (~200KB binary, 95% smaller!)

See [zig/README.md](zig/README.md) for details on the Zig implementation.

## Features

- **Multiple Log Sources**: Support for files, journalctl, dmesg, and custom command outputs
- **Pattern-based Detection**: Configurable regex patterns to identify issues
- **Sentry Integration**: Direct integration with Sentry for error tracking and alerting
- **Lightweight**: Minimal CPU and memory footprint
- **Flexible Configuration**: Command-line flags and environment variables
- **Real-time Monitoring**: Continuous monitoring with configurable check intervals

## Installation

### Prerequisites

- Go 1.19 or later
- Zig 0.11.0 or later (for Zig build)

### Building from Source

**Using Make (builds both Go and Zig versions):**

```bash
git clone https://github.com/angch/sentrylogmon.git
cd sentrylogmon

# Check if prerequisites are installed
make check-prereqs

# Install prerequisites if needed (downloads to /tmp)
make install-prereqs

# Build both Go and Zig binaries
make build

# Or build individually
make build-go   # Builds sentrylogmon (Go)
make build-zig  # Builds zig/zig-out/bin/sentrylogmon-zig
```

**Building Go version only:**

```bash
git clone https://github.com/angch/sentrylogmon.git
cd sentrylogmon
go build -o sentrylogmon
```

**Building Zig version only:**

```bash
git clone https://github.com/angch/sentrylogmon.git
cd sentrylogmon/zig
zig build -Doptimize=ReleaseSafe
# Binary will be at zig-out/bin/sentrylogmon-zig
```

### Installing via go install

```bash
go install github.com/angch/sentrylogmon@latest
```

## Usage

### Basic Usage

Monitor a log file for errors:

```bash
sentrylogmon --dsn="https://your-sentry-dsn@sentry.io/project" --file=/var/log/app.log
```

### Configuration Options

#### Sentry DSN

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

### systemd Service

Create `/etc/systemd/system/sentrylogmon.service`:

```ini
[Unit]
Description=Sentry Log Monitor
After=network.target

[Service]
Type=simple
User=nobody
Environment="SENTRY_DSN=https://your-dsn@sentry.io/project"
ExecStart=/usr/local/bin/sentrylogmon --file=/var/log/app.log --environment=production
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable sentrylogmon
sudo systemctl start sentrylogmon
sudo systemctl status sentrylogmon
```

## Development

### Building

Using Make (recommended):
```bash
# Build both Go and Zig versions
make build

# Build only Go
make build-go

# Build only Zig
make build-zig

# Build Zig with maximum size optimization
make build-zig-small

# Compare binary sizes
make compare-size
```

Building manually:
```bash
# Go version
go build -o sentrylogmon

# Zig version
cd zig && zig build
```

### Testing

```bash
# Go tests
make test-go
# or
go test ./...

# Zig tests
make test-zig
# or
cd zig && zig build test
```

### Linting

```bash
golangci-lint run
```

### Makefile Targets

Run `make help` to see all available targets:

```bash
make help
```

Available targets include:
- `make build` - Build both Go and Zig binaries
- `make clean` - Remove build artifacts
- `make check-prereqs` - Check if Go and Zig are installed
- `make install-prereqs` - Download and install prerequisites
- `make test` - Run tests
- `make compare-size` - Compare binary sizes

## Configuration File Support (Future)

Future versions may support configuration files for easier management of multiple monitors:

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
    pattern: "(?i)(error|critical)"
  
  - name: app-journal
    type: journalctl
    args: "--unit=myapp.service -f"
    pattern: "(?i)(error|fatal|panic)"
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
