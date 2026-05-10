## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-09 - CLI Table Empty States
**Learning:** Returning a bare table header or empty output when no items are present is confusing UX. Explicit empty states (e.g., "No running instances found") provide immediate clarity.
**Action:** Always check for empty states before rendering CLI table structures, particularly inside TTY-only branches to avoid breaking machine-readable JSON output.
