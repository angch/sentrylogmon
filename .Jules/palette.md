## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-21 - Status Table UX
**Learning:** Go's `text/tabwriter` aligns columns based on byte length, not visual width. Using multi-byte characters like emojis (e.g., 🔵) causes misalignment because their byte length (4) exceeds their visual width (2), shifting subsequent columns.
**Action:** Avoid emojis in `tabwriter` tables unless using a runewidth-aware library or manual padding. Use consistent ASCII or single-byte characters for reliable alignment.
