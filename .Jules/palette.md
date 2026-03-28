## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-28 - Scaffolding consistency across platforms
**Learning:** When adding micro-UX features like scaffolding starter configurations (`--init`), it is important to proactively propagate the feature across all language ports (Go, Rust, Zig) to maintain UX parity.
**Action:** Always check other implementations of a tool to ensure UX improvements are applied uniformly.
