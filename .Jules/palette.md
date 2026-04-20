## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-04-20 - CLI Status Output Empty State Parity
**Learning:** Empty states are critical for CLI usability. When outputting machine-readable data (like JSON), missing empty states are fine or even preferred, but for TTYs, printing bare headers with no data rows confuses users. Achieving parity across multiple implementations (Go, Rust, Zig) requires strict attention to TTY conditional logic.
**Action:** When printing tables or lists to a terminal, always include an explicit empty state check specifically within the TTY block to provide human-readable feedback without breaking machine output.
