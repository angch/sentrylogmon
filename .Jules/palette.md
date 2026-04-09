## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-13 - Empty States in CLI Actions
**Learning:** CLI commands that perform actions on collections (like `--update` restarting all running instances) should not fail silently or just do nothing when the collection is empty. Lack of feedback leaves the user wondering if the command worked or if the system is broken.
**Action:** Always provide explicit feedback for empty states in CLI actions (e.g., "No running instances found to update."), just as you would provide an empty state illustration or message in a graphical UI.
