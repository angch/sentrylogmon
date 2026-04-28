## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-04-28 - CLI Empty States
**Learning:** Empty states for CLI tables must be conditionally rendered only in TTY environments. If rendered in non-TTY (JSON/machine-readable) environments, human-readable strings break parsing for downstream tools.
**Action:** Always wrap empty state messages like "No results found" in TTY-checks when the command supports machine-readable output formats.
