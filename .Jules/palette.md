## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-05-06 - Helpful Empty States in CLI
**Learning:** Returning an empty table without feedback is confusing. Adding a "helpful empty state" (e.g., "No running instances found.") specifically in the human-readable TTY output significantly improves UX without breaking machine-readable JSON formats.
**Action:** Always check for empty collections before rendering complex UI elements like tables, providing clear text feedback to the user instead.
