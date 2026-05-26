## 2026-02-02 - [Smart List Truncation]
**Learning:** When truncating a list of items for a compact table view, blind character counting can hide all data if the first item is long or if the suffix " (+N more)" pushes the first item over the limit. A better UX guarantees at least the first item is shown (truncated if necessary) before summarizing the rest.
**Action:** Use an item-aware truncation loop that prioritizes the first item's visibility and reserves space for the summary suffix only for subsequent items.

## 2026-02-02 - [Human-Readable Durations]
**Learning:** For long-running processes, displaying uptime in raw hours (e.g., "50h") forces users to do mental math. Breaking it down into days (e.g., "2d 2h") respects the user's cognitive load and aligns with standard CLI patterns.
**Action:** Use a day-aware duration formatter for time spans >= 24 hours.

## 2026-05-26 - [CLI JSON Output Consistency]
**Learning:** When a CLI tool outputs JSON intended for programmatic consumption (like piping to `jq`), returning `null` for an empty result set breaks pipelines that expect an iterable array. This is a common DX failure when `nil` slices in Go are serialized.
**Action:** Always initialize slices returned as JSON arrays (e.g., `instances := []Type{}`) rather than leaving them as `nil`, ensuring the output is a valid empty array `[]`.
