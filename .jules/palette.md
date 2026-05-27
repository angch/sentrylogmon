## 2026-02-02 - [Smart List Truncation]
**Learning:** When truncating a list of items for a compact table view, blind character counting can hide all data if the first item is long or if the suffix " (+N more)" pushes the first item over the limit. A better UX guarantees at least the first item is shown (truncated if necessary) before summarizing the rest.
**Action:** Use an item-aware truncation loop that prioritizes the first item's visibility and reserves space for the summary suffix only for subsequent items.

## 2026-02-02 - [Human-Readable Durations]
**Learning:** For long-running processes, displaying uptime in raw hours (e.g., "50h") forces users to do mental math. Breaking it down into days (e.g., "2d 2h") respects the user's cognitive load and aligns with standard CLI patterns.
**Action:** Use a day-aware duration formatter for time spans >= 24 hours.

## 2026-05-27 - [Handling Empty States in CLI Output]
**Learning:** When a CLI application retrieves an empty list of items, printing an empty table or silently exiting is confusing. Explicitly stating that no items were found provides better feedback. This is a common UX pattern that should be consistent across all language implementations of a tool.
**Action:** Always provide clear, human-readable empty state messages (e.g., "No items found") when rendering lists or tables in TTY environments.
