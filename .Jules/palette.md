## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-03-09 - Ensure Polyglot CLI Consistency
**Learning:** When applying a UX improvement (like an `--init` flag) to a project with multiple language implementations (Go, Rust, Zig), it is critical to implement the improvement across all versions. Leaving out an implementation creates an inconsistent user experience and fails parity checks.
**Action:** Always check for other implementations of the same tool or component in the codebase, and apply UX changes uniformly. Use the codebase's validation scripts to ensure parity.
