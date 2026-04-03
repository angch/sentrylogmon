## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-26 - Scaffold Configuration for CLI Ports
**Learning:** Adding an `--init` flag to scaffolding configurations isn't just useful for the reference implementation, but must be available in all language ports (Rust, Zig) for a consistent user experience.
**Action:** When adding micro-UX features like starter config generation to a CLI tool, always propagate it across all versions of the application to maintain UX parity.
