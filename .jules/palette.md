## 2026-02-02 - [Smart List Truncation]
**Learning:** When truncating a list of items for a compact table view, blind character counting can hide all data if the first item is long or if the suffix " (+N more)" pushes the first item over the limit. A better UX guarantees at least the first item is shown (truncated if necessary) before summarizing the rest.
**Action:** Use an item-aware truncation loop that prioritizes the first item's visibility and reserves space for the summary suffix only for subsequent items.

## 2026-02-02 - [Human-Readable Durations]
**Learning:** For long-running processes, displaying uptime in raw hours (e.g., "50h") forces users to do mental math. Breaking it down into days (e.g., "2d 2h") respects the user's cognitive load and aligns with standard CLI patterns.
**Action:** Use a day-aware duration formatter for time spans >= 24 hours.
## 2026-06-06 - [Smart List Truncation for Zig Port]
**Learning:** Consistently applying standard UX patterns across different language ports is crucial for a cohesive user experience. The Zig port lacked the smart truncation logic introduced in the Go and Rust versions, leading to a degraded CLI experience.
**Action:** Aligned the Zig port's getDetails with the item-aware truncation loop from Go/Rust to guarantee at least the first item is shown before summarizing the rest.
