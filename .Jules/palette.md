## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-23 - Add Empty State for CLI Commands
**Learning:** For CLI tools that iterate over instances or print tables (like `--status` or `--update`), an explicit empty state message (e.g., "No running instances found.") improves user experience significantly compared to silent exits or bare table headers.
**Action:** When implementing list or update commands in CLI apps, always check for empty collections and provide helpful feedback to the user before proceeding.
