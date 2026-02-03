# Changelog of Major Decisions

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-27 | Initial design using Go + Sentry | Best balance of performance and integration capabilities |
| 2026-01-27 | CLI flags + env vars for config | Simplicity and container-friendliness |
| 2026-01-27 | Support for files, journalctl, dmesg | Cover 90% of common use cases |
| 2026-01-27 | Added `sysstat` for system context | Provide crucial context (load, memory) for debugging errors |
| 2026-01-27 | Ignore non-txt files in testdata | Prevent editor backups and artifacts from breaking data-driven tests |
| 2026-01-27 | Added Configuration File Support | Support for complex multi-monitor setups via YAML config |
| 2026-02-02 | User-isolated IPC directory | Security: Prevent local DoS/collision by using `/tmp/sentrylogmon-<uid>` (Unix) or per-user temp (Windows) |
