## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-05-09 - Empty State for CLI Tables
**Learning:** Rendering bare table headers or empty JSON arrays when there is no data creates a confusing "broken" appearance for CLI tools.
**Action:** Always verify data length and provide a clear, human-readable empty state (e.g., "No running instances found.") before initializing tabwriters or JSON encoders, especially guarding TTY-only behavior.
