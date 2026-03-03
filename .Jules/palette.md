## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-03 - Cross-Platform Empty State Parity
**Learning:** UX improvements shouldn't be siloed by language/implementation. While the Go version had a helpful `--init` empty state for configuration generation, the Rust and Zig ports lacked it, causing inconsistent onboarding friction depending on which binary the user downloaded.
**Action:** When working on multi-language CLI tools, ensure critical DX/UX features (like scaffolding/empty states) are ported to all implementations to maintain a consistent developer experience.
