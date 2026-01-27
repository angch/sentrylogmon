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

## Features

- **Multiple Log Sources**: Support for files, journalctl, dmesg, and custom command outputs
- **Pattern-based Detection**: Configurable regex patterns to identify issues
- **Sentry Integration**: Direct integration with Sentry for error tracking and alerting
- **System Status Context**: Automatically captures and attaches system state (CPU load, memory usage, top processes) to Sentry events
- **Lightweight**: Minimal CPU and memory footprint
- **Flexible Configuration**: Command-line flags and environment variables
- **Real-time Monitoring**: Continuous monitoring with configurable check intervals

## Installation

### Prerequisites

- Go 1.19 or later

### Building from Source

```bash
git clone https://github.com/angch/sentrylogmon.git
cd sentrylogmon
go build -o sentrylogmon
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

```bash
go build -o sentrylogmon
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

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
