## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-04-05 - Empty States Parity
**Learning:** When adding empty state feedback for CLI table outputs, it is easy to miss edge cases in multi-language ports, leaving users with confusing blank tables.
**Action:** When adding micro-UX features like empty state feedback to CLI tools, ensure the feature is proactively propagated across all language ports (Go, Rust, Zig) to maintain UX parity.
