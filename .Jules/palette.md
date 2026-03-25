## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-25 - Added empty state feedback to CLI commands
**Learning:** CLI lists and loops without explicit empty state messaging fail to provide users clear confirmation of system state.
**Action:** Add explicit empty state messages before loops when iterating over lists of instances.
