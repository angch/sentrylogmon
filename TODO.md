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

- [ ] **Implement Syslog Source**
  - Implement a `SyslogSource` struct (likely in `sources/syslog.rs`) that supports both UDP and TCP listeners.
  - Ensure it implements the `LogSource` trait.
  - Update `main.rs` to handle `--syslog` flag and corresponding config field.

- [x] **Update Status Output**
  - Improve the output of the `--status` command to match the detailed table format of the Go version.
  - Columns: `PID`, `STARTED`, `UPTIME`, `VERSION`, `DETAILS`.
  - Should include monitor names and types in the `DETAILS` column.
  - Detect TTY to decide whether to print table or JSON (if JSON output is desired for scripting, though Go defaults to table for TTY).

## Zig Parity

- [ ] **Implement Syslog Source**
  - Implement a syslog receiver (UDP/TCP) in `sources/syslog.zig` (or within `main.zig` if keeping single file structure, though a separate file is preferred).
  - Update `parseArgs` and configuration loading to support syslog.

- [ ] **Update Status Output**
  - Update `main.zig`'s status printing logic to match Go's table format.
  - Columns: `PID`, `STARTED`, `UPTIME`, `VERSION`, `DETAILS`.
  - Calculate uptime from start time.
  - Format details to show monitor summary.

## General

- [ ] **Profile Memory Usage**
  - Run the application with `pprof` (Go) or Valgrind/Massif (Rust/Zig) under load to identify memory bottlenecks.
