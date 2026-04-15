## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-02 - [CLI Empty State Feedback]
**Learning:** When outputting tables in a CLI without data rows, simply showing empty headers leads to confusion and poor discoverability. Explicit empty states are required to proactively confirm that the system correctly evaluated there is no active data.
**Action:** Always add `if (items.len == 0)` checks before drawing CLI tables, proactively echoing "No [items] found." to guide the user.
