# AGENTS.md - Guide for LLM Agents

This document indexes resources and guidelines for agents working on `sentrylogmon`.

## Documentation

- **Design Principles**: `doc/DESIGN.md` (Motivation, Solution, Design Choices)
- **Architecture**: `doc/ARCHITECTURE.md` (Structure, Abstractions, Tech Stack)
- **Testing**: `doc/TESTING.md` (Unit, Data-Driven, Integration, Performance)
- **Development**: `doc/DEVELOPMENT.md` (Common Pitfalls)
- **History**: `doc/HISTORY.md` (Changelog of Major Decisions)
- **Roadmap**: `doc/ROADMAP.md` (Planned Features, Non-Goals)

## Guides

- **Contributing**: `CONTRIBUTING.md` (Guidelines for Agents)
- **Rust Development**: `rust/README.md` (See "Agent Development Guidelines" section)
- **Go Style**: `.agent/skills/go/STYLE.md`

## Resources

- **Go Best Practices**: https://go.dev/doc/effective_go
- **Sentry Go SDK Docs**: https://docs.sentry.io/platforms/go/
- **12-Factor App**: https://12factor.net/
- **Unix Philosophy**: https://en.wikipedia.org/wiki/Unix_philosophy

## Performance Profile (2026-02-04)

### Memory Profile Analysis
A heap profile was captured under load (100,000 log lines) using `net/http/pprof`.

- **Total Memory Usage**: ~9MB in-use during high load test.
- **Top Consumers**:
  - `bytes.growSlice` (53%): Primarily driven by `github.com/getsentry/sentry-go` during envelope creation and event buffering.
  - `crypto/tls` & `encoding/pem` (~11%): SSL/TLS handshake overhead for Sentry connections.
  - `regexp` (~6%): Timestamp extraction in `monitor.extractTimestamp`.
- **Conclusion**: The application is memory efficient. Most allocation comes from the Sentry SDK's necessary buffering and transmission logic. No obvious leaks or inefficiencies in the application code were found.

---

**Last Updated**: 2026-02-04

## Agent Notes

- 2026-02-04: Rust `--status` table output now uses dynamic column widths and a deterministic formatter (`format_instance_table`). The helper relies on fixed-width start timestamps (`%Y-%m-%d %H:%M:%S`) and aligns the DETAILS column based on max widths. Tests validate alignment by matching DETAILS column offsets rather than exact timestamps.
- 2026-02-04: Zig `--status` output updated to use `TabWriter` for dynamic column alignment, matching Go behavior. Implementation notes:
  - Zig 0.15.2 `std.fs.File.stdout().writer()` requires a buffer argument and returns a struct where the `Writer` interface is accessed via `.interface` field.
  - `std.ArrayList` behaves like `std.ArrayListUnmanaged` in this version (requires explicit allocator for methods).
- 2026-02-04: Zig implementation now exposes `/debug/pprof/` endpoints for parity. `cmdline` is fully implemented, while `profile`, `heap`, etc., return 501 Not Implemented with instructions to use native tools (perf, Valgrind) as Zig lacks a standard runtime profiler.
- 2026-02-05: `JsonDetector` optimized with caching.
  - **Key Decision**: Detectors can be stateful but must be thread-safe (`sync.Mutex`) and validate cache against input (`bytes.Equal`) because `GetContext` can be called with arbitrary lines.
  - **Performance**: Safe caching (Mutex + Copy + Compare) yielded ~50% speedup over double unmarshalling (2900ns vs 6100ns).
