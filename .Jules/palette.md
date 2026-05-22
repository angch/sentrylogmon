## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2024-05-22 - Empty JSON Array Output
**Learning:** Returning `null` instead of `[]` for empty sets in JSON APIs/CLIs breaks downstream tools like `jq` and causes poor developer experience. Initializing slices correctly (`[]Type{}`) fixes this.
**Action:** When outputting JSON arrays, always ensure empty collections serialize to `[]` instead of `null` for better tool interoperability and UX.
