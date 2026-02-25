## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-25 - CLI Table Clarity
**Learning:** Hardcoding 'Running' status in `status` commands might seem redundant, but it provides explicit confirmation and consistent layout with other system tools (like `docker ps`), reducing cognitive load.
**Action:** Ensure status commands always include a STATUS column, even if the value is static for now.
