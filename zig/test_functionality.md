# Zig Implementation Testing Guide

This document describes how to test the Zig implementation of sentrylogmon.

## Prerequisites

Ensure Zig 0.13.0 or later is installed:

```bash
zig version
```

If not installed, see the main README or run:
```bash
make install-prereqs
```

## Building the Zig Binary

```bash
# From the repository root
make build-zig

# Or from the zig directory
cd zig
zig build -Doptimize=ReleaseSafe
```

The binary will be created at `zig/zig-out/bin/sentrylogmon-zig`.

## Manual Testing

### Test 1: Help and Usage

Test that the binary requires proper arguments:

```bash
# Should fail with error message
./zig-out/bin/sentrylogmon-zig

# Expected output:
# Sentry DSN is required. Set via --dsn flag or SENTRY_DSN environment variable
```

### Test 2: File Monitoring

Create a test log file:

```bash
cat > /tmp/test.log << 'EOF'
[1234.5678] Normal log line
[1234.5679] Error: something went wrong
[1234.5680] Another line
[1235.0001] Error: different error
[1236.0000] Fatal error in module
EOF
```

Run the Zig binary (with a test DSN):

```bash
export SENTRY_DSN="https://test@sentry.io/123"
./zig-out/bin/sentrylogmon-zig --file /tmp/test.log --verbose
```

Expected behavior:
- Should read the file
- Should detect lines containing "Error" (default pattern)
- Should group by timestamp
- Should attempt to send to Sentry (will fail with test DSN, but that's OK)

### Test 3: Custom Pattern Matching

```bash
./zig-out/bin/sentrylogmon-zig --file /tmp/test.log --pattern="Fatal" --verbose
```

Expected behavior:
- Should only match the "Fatal error" line
- Should group and attempt to send to Sentry

### Test 4: Environment Variables

```bash
export SENTRY_DSN="https://test@sentry.io/123"
./zig-out/bin/sentrylogmon-zig --file /tmp/test.log --environment=testing --release=v1.0.0
```

Expected behavior:
- Should use DSN from environment
- Should tag events with environment=testing and release=v1.0.0

### Test 5: dmesg Monitoring

```bash
# Requires root or appropriate permissions
sudo ./zig-out/bin/sentrylogmon-zig --dmesg --pattern="error" --verbose
```

Expected behavior:
- Should run `dmesg -w` command
- Should monitor kernel messages
- Should detect errors in kernel log

### Test 6: Specific Log Formats

Test nginx-error format:

```bash
# Create test file
cat > /tmp/nginx-error.log << 'EOF'
2026/01/27 10:00:00 [error] 1234#0: *1 connect() failed (111: Connection refused)
2026/01/27 10:00:01 [info] 1234#0: *2 client closed connection
EOF

# Run test
./zig/zig-out/bin/sentrylogmon-zig --file /tmp/nginx-error.log --format=nginx-error --verbose --oneshot
```

Expected behavior:
- Should detect the "[error]" line
- Should NOT detect the "[info]" line

## Comparison Testing

Compare output and behavior with the Go version:

```bash
# Test Go version
export SENTRY_DSN="https://test@sentry.io/123"
./sentrylogmon --file /tmp/test.log --verbose

# Test Zig version
./zig/zig-out/bin/sentrylogmon-zig --file /tmp/test.log --verbose
```

Both should:
- Detect the same error lines
- Group by the same timestamps
- Produce similar output (allowing for formatting differences)

## Binary Size Comparison

```bash
make compare-size
```

Expected results:
- Go binary: ~8-10 MB
- Zig binary (ReleaseSafe): ~200-400 KB
- Zig binary (ReleaseSmall): ~100-200 KB

The Zig binary should be approximately 95% smaller.

## Performance Testing

### Memory Usage

```bash
# Test Go version
/usr/bin/time -v ./sentrylogmon --file /tmp/test.log 2>&1 | grep "Maximum resident"

# Test Zig version
/usr/bin/time -v ./zig/zig-out/bin/sentrylogmon-zig --file /tmp/test.log 2>&1 | grep "Maximum resident"
```

Expected: Zig version should use less memory.

### Startup Time

```bash
# Test Go version
time ./sentrylogmon --file /tmp/test.log

# Test Zig version
time ./zig/zig-out/bin/sentrylogmon-zig --file /tmp/test.log
```

Expected: Zig version should have faster startup.

## Integration Testing with Real Sentry

**Warning**: This will send real events to Sentry.

1. Get a real Sentry DSN from your Sentry project
2. Set it as an environment variable:
   ```bash
   export SENTRY_DSN="https://your-real-key@sentry.io/your-project"
   ```
3. Run the Zig binary:
   ```bash
   ./zig/zig-out/bin/sentrylogmon-zig --file /tmp/test.log --verbose
   ```
4. Check your Sentry project for incoming events
5. Verify events have:
   - Correct message: "Log errors at timestamp [...]"
   - Tags: timestamp, source
   - Extra data: log_lines with timestamp, line_count, lines

## Known Limitations

The Zig version has these differences from the Go version:

1. **Pattern Matching**: Uses case-insensitive substring matching instead of full regex
   - Go: Full regex support
   - Zig: Simple substring search (faster, but less flexible)

2. **HTTP Client**: Direct HTTP calls instead of Sentry SDK
   - Go: Uses official Sentry SDK
   - Zig: Manual HTTP POST to Sentry API

3. **Error Handling**: Different error message formats
   - Both versions handle errors, but messages may differ slightly

## Troubleshooting

### "Zig not found"

Ensure Zig is in your PATH:
```bash
export PATH=/path/to/zig:$PATH
```

### "Failed to open file"

Ensure the log file exists and is readable:
```bash
ls -l /tmp/test.log
```

### "Could not resolve host"

Check network connectivity:
```bash
ping sentry.io
```

### Build errors

Ensure you're using Zig 0.13.0 or later:
```bash
zig version
```

## Automated Testing

Currently, the Zig implementation doesn't have unit tests (coming soon).

To add unit tests:

```bash
cd zig
zig build test
```

## Success Criteria

The Zig implementation is working correctly if:

1. ✅ It builds without errors
2. ✅ It parses command-line arguments correctly
3. ✅ It reads log files and detects patterns
4. ✅ It groups log lines by timestamp
5. ✅ It constructs valid Sentry API payloads
6. ✅ It sends HTTP POST requests to Sentry
7. ✅ Binary size is significantly smaller than Go version
8. ✅ Memory usage is lower than Go version
9. ✅ Functionally equivalent to Go version (with documented limitations)
