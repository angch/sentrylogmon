## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-07 - Command List Empty States
**Learning:** Commands that list items (like `--status` or `--update`) need graceful empty states.
**Action:** Always verify empty states and print a helpful message (e.g., 'No running instances found.') instead of empty table headers or silently exiting when outputting to a TTY.
