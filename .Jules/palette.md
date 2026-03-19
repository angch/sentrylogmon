## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-19 - CLI Empty States
**Learning:** When performing CLI operations that list or iterate over running instances (like `--status` or `--update`), executing without output when no instances exist creates confusion. Users are left wondering if the command failed silently or succeeded with no targets.
**Action:** Always provide an explicit empty state message (e.g. "No running instances found.") when a list or action target array is empty, ensuring clear feedback in both TTY and non-TTY contexts.
