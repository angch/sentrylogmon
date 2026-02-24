## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-07 - CLI Responsiveness
**Learning:** CLI tools often assume 80-column terminals, leading to wasted space on modern screens or broken layouts on narrow ones. Detecting terminal width allows for responsive text truncation that feels "native" and maximizes information density.
**Action:** When designing CLI tables or lists, always detect terminal width and adjust column widths dynamically instead of using hardcoded limits.
