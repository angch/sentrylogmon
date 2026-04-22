## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-04-22 - Empty States in CLI Tables
**Learning:** When outputting tabular data in CLI tools, rendering a header row with no data is confusing. Implementing an explicit empty state check (e.g. "No items found") prevents visual noise and improves the onboarding/empty experience. Crucially, this must only apply to human-readable TTY output to avoid breaking machine-readable formats.
**Action:** Always check for empty collections before drawing table headers in CLI tools, specifically within interactive TTY conditions.
