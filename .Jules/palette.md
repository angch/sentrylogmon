## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-13 - Empty States in CLI Iteration Commands
**Learning:** Empty states are just as important for CLI tools as they are for GUIs. When commands that list (`--status`) or iterate over items (`--update`) find nothing, silently succeeding or printing an empty table header looks like a bug.
**Action:** Always verify what happens when iterating over an empty list in CLI tools and add explicit feedback (e.g., "No running instances found.") to assure the user the command worked but had nothing to process.
