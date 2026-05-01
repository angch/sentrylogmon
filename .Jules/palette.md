## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-08 - Consistent Empty States for Machine/Human Output
**Learning:** Empty states are critical for both human readers (TTY) and machine parsers (JSON). Failing to initialize an empty list properly in Go causes JSON serialization to output `null` instead of `[]`, breaking downstream `jq` parsers, while failing to provide a human-readable empty state in Zig leaves users with a confusing header-only table. Both scenarios represent poor UX/DX.
**Action:** When implementing empty states, explicitly test both interactive (TTY) and non-interactive (JSON/machine) outputs to ensure consistent, expected formats.
