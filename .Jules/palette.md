## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-14 - Empty State CLI Output
**Learning:** When displaying structured tabular data in CLI (like `--status`), it's confusing to show an empty table header when no items exist.
**Action:** Always add an explicit empty state check (e.g., 'No running instances found') before rendering table headers.
