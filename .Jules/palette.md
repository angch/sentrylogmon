## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-06-26 - CLI Empty State UX
**Learning:** For CLI tools that run as daemons or long-running processes (like sentrylogmon), printing an error about a missing configuration and exiting immediately on `len(os.Args) == 1` is a poor user experience. Users who run the bare command without arguments often just want to see what the tool does or how to use it. Automatically showing the usage guide (`flag.Usage()`) provides a helpful "empty state" instead of an aggressive error.
**Action:** When a CLI tool is executed without any arguments and fails to load a default configuration, catch this empty state and print the usage guide before exiting, turning a frustrating failure into a helpful onboarding moment.
