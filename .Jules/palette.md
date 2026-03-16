## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-23 - Add empty state feedback for CLI status and update commands
**Learning:** CLI tools often fail silently or print empty table headers when attempting to iterate over items or display statuses for empty collections. This leads to poor UX.
**Action:** When creating or modifying CLI tools, always check if collections are empty before iterating over them or printing empty table headers, and output an explicit "No items found." or similar empty state message.
