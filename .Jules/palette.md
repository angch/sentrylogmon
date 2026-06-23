## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-06-23 - CLI Empty State UX
**Learning:** When a CLI daemon is executed with no arguments and fails due to missing configuration, displaying the full usage guide along with a helpful tip (like using `--init`) provides a much better onboarding experience than a terse validation error.
**Action:** Intercept configuration validation failures specifically when no arguments are provided, and print the usage guide and onboarding tips instead of a raw error.
