## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-10 - CLI Empty States Feedback
**Learning:** Commands that iterate over items (like `--update`) should not fail or exit silently when the item list is empty. Users may assume the command failed, froze, or had a bug.
**Action:** Always provide explicit "No [items] found" messages for list-based CLI operations to confirm the command ran successfully but found nothing to do.
