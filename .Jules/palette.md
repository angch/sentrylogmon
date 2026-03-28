## 2026-02-07 - CLI Onboarding UX
**Learning:** UX isn't just for GUIs. Adding an `--init` flag to generate a starter configuration file significantly reduces onboarding friction for CLI tools, acting as a "helpful empty state."
**Action:** For CLI tools with complex configuration, always look for ways to scaffold the initial setup (e.g., `init` commands, interactive wizards).

## 2026-02-09 - CLI Table Readability
**Learning:** For CLI table outputs, exact precision (e.g., "1d 2h 30m 15s") is often less useful than clean, concise approximations (e.g., "1d 2h"). Reducing cognitive load by hiding lower-order time units makes the important data stand out.
**Action:** When displaying durations or sizes in a summary view, prioritize readability and conciseness over maximum precision. Use `text/tabwriter` for alignment instead of manual padding.
