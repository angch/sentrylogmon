## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-06-29 - Improve CLI daemon empty-state UX
**Learning:** For CLI daemons, automatically printing usage and exiting when no arguments are provided is incorrect, as they often run workloads via default config files.
**Action:** Only print the usage guide if the configuration fails to load or validate, and ensure the actual configuration error message is not swallowed by the exit.
