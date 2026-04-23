## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2023-10-28 - Improve empty state feedback for CLI output
**Learning:** Returning null for empty arrays or just an empty table header without context leaves users confused about whether the command worked but found nothing or if an error occurred in parsing. Empty states need explicit contextual feedback (e.g. "No instances found" or a valid empty JSON array `[]`) to be machine- and human-readable.
**Action:** When printing tables or JSON arrays, ensure an empty state check explicitly provides "No [items] found." or `[]` before generating default headers.
