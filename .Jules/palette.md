## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-08 - CLI Empty State UX
**Learning:** Adding empty state feedback for CLI table outputs (e.g., when no instances are running) is crucial for a helpful user experience, confirming the command worked but there's just no data.
**Action:** Always verify if a CLI table has an empty state check inside the TTY-only branch to prevent human-readable warnings from disrupting machine-readable output formats like JSON.
