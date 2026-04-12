## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-04-12 - Empty States in CLI Tables
**Learning:** When generating CLI table output, always check for empty data sets before printing headers. Printing only headers for empty data feels broken.
**Action:** Implement an explicit empty state check (e.g., "No running instances found.") before scaffolding table headers, ensuring this human-readable feedback is contained within TTY-only branches so it doesn't corrupt machine-readable outputs like JSON.
