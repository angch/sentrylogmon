## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-21 - Add explicit empty state to update command
**Learning:** Action commands that iterate over processes or resources need explicit empty state feedback just as much as list/status commands do. Silent exits when zero items are found can make users think the command failed or hung.
**Action:** When implementing any command that performs an action on a collection (e.g., --update, --restart, --delete), explicitly handle the `len == 0` case with a clear message like 'No items found' instead of a silent no-op.
