## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-02-07 - CLI Empty State UX
**Learning:** For CLI tools that output formatted tables or lists, failing silently or returning empty outputs can leave users confused about whether a command failed or successfully found nothing.
**Action:** Always provide explicit, human-readable empty state feedback (e.g., "No running instances found.") when outputting to a TTY, while ensuring machine-readable formats like JSON remain unaffected.
