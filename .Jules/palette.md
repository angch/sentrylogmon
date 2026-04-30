## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-05-01 - Empty States
**Learning:** Tables representing lists of items should always have an empty state fallback. Otherwise, users are left wondering if the command failed or if the output is just empty (especially when headers are also suppressed or missing). For CLI table outputs (like `--status`), if there are no instances, displaying a clear "No running instances found." message provides much better UX than silently returning or printing only headers.
**Action:** Always verify empty states for list/table outputs in CLIs, specifically checking the TTY branch to ensure machine-readable formats (JSON) remain unaffected.
