## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-08 - CLI Empty States UX
**Learning:** Returning an empty array without context for a CLI table command looks broken. Providing an explicit empty state string in TTY mode improves readability.
**Action:** Always add human-readable empty state feedback for commands that normally output tables, ensuring the check happens only in the TTY branch.
