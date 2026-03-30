## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2024-05-24 - Empty State Feedback for Action Commands
**Learning:** Action commands that iterate over items (like `--update` or `--status`) fail silently or present blank states to the user if the collection is empty.
**Action:** Always add explicit, helpful empty state feedback (e.g., "No running instances found.") before iterations to improve micro-UX and system visibility.
