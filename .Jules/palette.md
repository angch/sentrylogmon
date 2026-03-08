## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-08 - Empty State Formatting in Lists
**Learning:** Commands that process lists of items silently (like a restart or update command) create confusion when the list is empty because the user gets no output.
**Action:** Always provide explicit, helpful feedback (e.g., "No running instances found.") when an operation is executed on an empty state or list, rather than exiting silently.
