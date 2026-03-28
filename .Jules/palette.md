## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-02 - Empty States in CLI Tools
**Learning:** Empty states are just as important in CLI tools as GUIs. For example, printing just a table header when a command like `sentrylogmon-zig --status` returns no items is poor UX compared to a helpful message like "No running instances found."
**Action:** Always test the empty or "no results" state of CLI outputs and provide explicit feedback instead of blank tables.
