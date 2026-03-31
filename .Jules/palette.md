## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-24 - Empty State Feedback for Actionable Flags
**Learning:** Actionable flags that iterate over a collection (like `--update`) or list items (like `--status`) need explicit empty state feedback to prevent silent failures and user confusion.
**Action:** Always verify loops over collections have an explicit check and feedback before executing when working with CLI tools.
