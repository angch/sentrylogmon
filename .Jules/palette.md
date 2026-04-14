## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).
## 2024-04-14 - Clear Empty States for CLI Tables
**Learning:** Without explicit empty states, CLI tables simply render orphaned headers which degrades developer experience. However, this human-readable empty state must ONLY be rendered on TTY outputs to avoid breaking machine-readable JSON outputs expected by automation scripts.
**Action:** Always intercept empty collections BEFORE table headers are printed, and strictly isolate this feedback within interactive terminal branches (e.g., `isTty()`).
