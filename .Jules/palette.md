## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-04-10 - Empty State Feedback for APIs and CLIs
**Learning:** CLI tools often act as APIs when piped. Empty states must be tailored to the medium: TTYs require human-readable text ("No running instances found.") to avoid confusing blank tables, while piped outputs require structurally valid representations (like `[]` instead of `null`) to prevent breaking downstream parsers.
**Action:** Always explicitly initialize empty collections before JSON serialization and test empty states in both TTY and piped contexts.
