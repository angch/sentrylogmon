## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-05-23 - JSON Output Null State
**Learning:** When outputting JSON arrays from a CLI using `json.Encoder`, ensuring empty slices are explicitly initialized (e.g., `instances := []Type{}`) rather than left as `nil` (`var instances []Type`) prevents the encoder from outputting `null`. This maintains compatibility with downstream JSON parsers like `jq` that expect an array, preventing surprising pipeline breaks.
**Action:** Always explicitly initialize slices meant to be serialized to JSON, especially for commands designed to be piped or used in scripts.
