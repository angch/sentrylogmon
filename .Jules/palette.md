## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-09 - Empty State for CLI Tables
**Learning:** When a CLI table is empty, displaying only headers creates confusion. Users might think the program hung or data is missing.
**Action:** Always implement empty state checks before initializing tabwriters or printing headers to provide clear "No data found" feedback.
