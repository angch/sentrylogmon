## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-09 - Empty States in Machine-Readable Commands
**Learning:** Missing empty states (like "No instances found") in CLI tools can leave users wondering if a command silently failed. However, empty state messages must only be output in TTY environments. Emitting human-readable text in non-TTY mode breaks downstream JSON parsers like `jq`.
**Action:** Always guard empty state feedback behind a TTY check, ensuring machine-readable output remains valid (e.g. `[]`).
