## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2024-07-07 - Add usage guide to empty-state configuration errors
**Learning:** When users run a CLI tool without arguments or with invalid configuration, they are often left with just a cryptic error message. Printing the usage guide along with the error significantly improves the empty-state experience and reduces user frustration by providing immediate context on how to correctly invoke the tool.
**Action:** Always ensure that CLI tools provide a usage guide or help text alongside configuration-related startup errors, rather than just terminating.
