## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-04-26 - CLI JSON Empty State
**Learning:** Returning a literal `null` for empty slices in JSON output from CLIs breaks downstream parsing tools (like `jq`), whereas outputting `[]` prevents pipeline failures and matches user expectations for list outputs.
**Action:** When a CLI command guarantees an array-like output (e.g. `--status`), always output an empty array `[]` instead of `null` when the underlying list is empty.
