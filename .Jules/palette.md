## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-06-17 - CLI Configuration Error UX
**Learning:** For CLI daemons, automatically exiting when no arguments are provided breaks default execution paths that rely on default configuration files. A better approach is to print the usage guide when the configuration explicitly fails to load or validate.
**Action:** When improving CLI empty states, do not rely on `len(os.Args) == 1`. Instead, catch configuration or validation errors and append the usage guide to the error output to guide the user without breaking expected daemon behavior.
