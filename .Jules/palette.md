## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-18 - Missing empty states in CLI outputs
**Learning:** The CLI action commands (`--update`, `--status`) in the Zig implementation can fail silently or print empty tables/JSON brackets `[]` when no instances are running. This creates confusion. The Go/Rust versions also had similar issues or print empty states for update without instances. An explicit "No running instances found." empty state is better UX and consistent with standard CLI conventions.
**Action:** When implementing CLI loops or lists, always check length and provide a helpful empty state message instead of silent completion or bare structure printing.
