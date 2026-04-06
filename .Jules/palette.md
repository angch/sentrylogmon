## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-07 - CLI Initialization Feedback
**Learning:** When scaffolding configuration files via an `--init` flag, providing explicit success messages (e.g., 'Generated file.yaml') and non-destructive overwrite warnings (e.g., 'File already exists') is critical for preventing user anxiety and errors.
**Action:** Always include safe existence checks and clear console feedback when writing generated files in CLI tools.
