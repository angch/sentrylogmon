## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-07 - CLI JSON Empty States
**Learning:** When outputting JSON from a CLI tool for automation/piping, returning `null` for empty collections breaks downstream parsers expecting arrays. Returning `[]` improves API accessibility.
**Action:** Always ensure JSON outputs initialize empty slices or manually print `[]` instead of `null` when no data exists.
