## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-14 - Empty States for CLI Action Commands
**Learning:** Action commands that iterate over items (like `--update` or `--status`) can silently exit or produce confusing outputs (like `null`) if the list is empty. Explicitly handling empty states is critical for good CLI UX.
**Action:** Always verify that commands interacting with lists of items explicitly print a helpful empty state message (e.g., "No running instances found.") or a valid empty array structure (`[]` for JSON) when no items exist.
