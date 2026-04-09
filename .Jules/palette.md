## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-09 - [Empty State Feedback]
**Learning:** Action commands that iterate over items need explicit empty states, otherwise the user receives no feedback when zero items are found, which feels like the command silently failed or hung.
**Action:** Always provide clear "Not found" or "No items" messaging when a list-based action has zero targets.
