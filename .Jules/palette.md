## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-05-12 - Empty State in CLI Tables
**Learning:** Returning an empty table header with no rows (e.g., `PID STARTED UPTIME...`) in a CLI is confusing because users must interpret the absence of data. Explicitly stating "No instances found" provides immediate, unambiguous feedback.
**Action:** When printing tables or lists to a TTY, always check for an empty collection first and print a clear, human-readable empty state message before printing headers.
