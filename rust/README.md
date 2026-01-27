# sentrylogmon - Rust Implementation

This is a lightweight Rust port of the sentrylogmon Go application. It provides the same functionality with an emphasis on being smaller and more resource-efficient.

## Features

- **Minimal Resource Footprint**: Optimized for size and memory usage
- **Async I/O**: Built on Tokio for efficient non-blocking operations
- **Multiple Log Sources**: Support for files, journalctl, dmesg, and custom commands
- **Pattern Matching**: Regex-based issue detection
- **Sentry Integration**: Direct integration with Sentry for error tracking
- **System Context**: Captures CPU, memory, and process information

## Building

### Prerequisites

- Rust 1.70 or later
- Cargo (comes with Rust)

### Build Commands

```bash
# Debug build
cargo build

# Release build (optimized for size)
cargo build --release
```

The release build is optimized for minimal binary size using:
- Link Time Optimization (LTO)
- Size optimization (`opt-level = "z"`)
- Symbol stripping
- Single codegen unit

## Usage

The Rust version supports the same command-line interface as the Go version:

```bash
# Monitor a log file
./target/release/sentrylogmon --dsn="https://your-dsn@sentry.io/project" --file=/var/log/app.log

# Monitor journalctl
./target/release/sentrylogmon --dsn="..." --journalctl="--unit=myapp.service -f"

# Monitor dmesg
./target/release/sentrylogmon --dsn="..." --dmesg

# Use a configuration file
./target/release/sentrylogmon --config=sentrylogmon.yaml
```

## Configuration

The Rust implementation uses the same YAML configuration format as the Go version:

```yaml
sentry:
  dsn: https://your-dsn@sentry.io/project
  environment: production
  release: v1.0.0

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

## Differences from Go Version

### Size and Performance

The Rust version is optimized for:
- Smaller binary size (release builds are typically smaller)
- Lower memory footprint
- Efficient async I/O with Tokio

### Architecture

- **Async/Await**: Uses Tokio's async runtime instead of goroutines
- **Trait-based**: LogSource and Detector are implemented as traits
- **Type Safety**: Leverages Rust's strong type system and ownership model

### Simplified Features

The Rust port focuses on core functionality:
- Generic pattern matching (specialized detectors can be added)
- Essential system stats collection
- Core log source types

## Development

### Running Tests

```bash
cargo test
```

### Code Formatting

```bash
cargo fmt
```

### Linting

```bash
cargo clippy
```

## Binary Size Comparison

The Rust release build is typically smaller than the equivalent Go binary:

```bash
# Check binary size
ls -lh target/release/sentrylogmon

# Strip additional symbols (if not already stripped)
strip target/release/sentrylogmon
```

## License

MIT License - same as the main project
