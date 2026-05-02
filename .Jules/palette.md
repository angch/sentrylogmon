## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-07 - CLI Empty States
**Learning:** Empty states are essential UX even in CLI environments. Rendering headers without data gives confusing feedback. Returning a clean "No running instances found." message provides helpful guidance.
**Action:** Always check for empty lists/arrays before rendering tabular data or list headers in CLI tools, and display a human-readable fallback message in TTY environments.
