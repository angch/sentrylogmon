## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2026-03-29 - CLI Onboarding UX Parity
**Learning:** Feature parity across language implementations is crucial for consistent UX. The `--init` flag existed in Go but was missing in Rust and Zig, breaking the promised onboarding experience for users choosing smaller binaries.
**Action:** When migrating or maintaining CLI tools across multiple languages, ensure setup/onboarding flags like `--init` are implemented universally to maintain a unified user experience.
