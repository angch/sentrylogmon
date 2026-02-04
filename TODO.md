# TODO - Work Items

This file tracks active work items to bring documentation and implementations (Rust, Zig) to parity with the Go reference implementation.

## Documentation Sync

- [x] **Update Go Documentation**
  - Update `README.md` to explicitly list **Syslog** (UDP/TCP) as a supported log source in the "Log Sources" section.

- [x] **Update Rust Documentation**
  - Update `rust/README.md` to note that **Syslog** source is currently missing/planned.

- [x] **Update Zig Documentation**
  - Update `zig/README.md` to note that **Syslog** source is currently missing.
  - Update `zig/README.md` to correctly reflect implemented features: **Journalctl**, **Configuration File**, and **Batching**.

## Rust Parity

- [x] **Implement Syslog Source**
  - Implement a `SyslogSource` struct (likely in `sources/syslog.rs`) that supports both UDP and TCP listeners.
  - Ensure it implements the `LogSource` trait.
  - Update `main.rs` to handle `--syslog` flag and corresponding config field.

- [x] **Update Status Output**
  - Improve the output of the `--status` command to match the detailed table format of the Go version.
  - Columns: `PID`, `STARTED`, `UPTIME`, `VERSION`, `DETAILS`.
  - Should include monitor names and types in the `DETAILS` column.
  - Detect TTY to decide whether to print table or JSON (if JSON output is desired for scripting, though Go defaults to table for TTY).

- [x] **Implement --format CLI Argument**
  - Add `--format` argument to `Args` struct in `config/mod.rs`.
  - Pass this format to the detector logic to override defaults (parity with Go).

- [ ] **Implement Metrics Server**
  - Add `--metrics-port` CLI argument and `MetricsPort` field to `Config`.
  - Implement a simple HTTP server to expose Prometheus metrics at `/metrics`.
  - Implement `/healthz` endpoint returning 200 OK.

- [ ] **Improve Status Output Alignment**
  - Use a dynamic table formatter (like `tabwriter` crate or manual padding calculation) to match Go's alignment behavior, ensuring columns don't break with long values.

- [ ] **Implement pprof Support**
  - Investigate and implement pprof-compatible endpoints (optional, for parity).

## Zig Parity

- [x] **Implement Syslog Source**
  - Implement a syslog receiver (UDP/TCP) in `sources/syslog.zig` (or within `main.zig` if keeping single file structure, though a separate file is preferred).
  - Update `parseArgs` and configuration loading to support syslog.

- [x] **Update Status Output**
  - Update `main.zig`'s status printing logic to match Go's table format.
  - Columns: `PID`, `STARTED`, `UPTIME`, `VERSION`, `DETAILS`.
  - Calculate uptime from start time.
  - Format details to show monitor summary.

- [ ] **Implement TTY Detection**
  - Update `status` command to check if `stdout` is a terminal (isatty).
  - If not a terminal, output JSON instead of the table (parity with Go).

- [ ] **Implement Metrics Server**
  - Add `--metrics-port` argument and config support.
  - Implement an HTTP server (using `std.http.Server` or similar) to expose `/metrics` and `/healthz`.

- [ ] **Improve Status Output Alignment**
  - Implement tab-writer style alignment for the status table instead of relying on simple tabs `\t`, to ensure consistent column spacing.

- [ ] **Implement pprof Support**
  - Investigate and implement pprof-compatible endpoints (optional, for parity).

## General

- [x] **Profile Memory Usage**
  - Run the application with `pprof` (Go) or Valgrind/Massif (Rust/Zig) under load to identify memory bottlenecks.
