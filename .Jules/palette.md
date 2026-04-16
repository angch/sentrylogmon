## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2025-02-09 - CLI Empty States
**Learning:** CLI applications should return clear human-readable empty state messages ("No running instances found.") in TTY environments, but must fall back to valid empty data structures (like `[]`) in non-TTY environments to preserve machine parsability.
**Action:** When implementing empty states for CLI lists, always wrap the human-readable warning in a TTY check.
