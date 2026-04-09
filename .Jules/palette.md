## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-15 - Graceful Empty States in CLI Actions
**Learning:** In CLI tools, action commands that iterate over lists (like `--update` restarting all instances) must explicitly handle empty states. Without explicit feedback, the command silently exits, leaving users uncertain if the action succeeded or if there was simply nothing to do.
**Action:** Always verify empty list conditions and provide clear, reassuring feedback (e.g., "No items found to process") before exiting.
