## 2026-02-02 - [Smart List Truncation]
**Learning:** When truncating a list of items for a compact table view, blind character counting can hide all data if the first item is long or if the suffix " (+N more)" pushes the first item over the limit. A better UX guarantees at least the first item is shown (truncated if necessary) before summarizing the rest.
**Action:** Use an item-aware truncation loop that prioritizes the first item's visibility and reserves space for the summary suffix only for subsequent items.

## 2026-02-02 - [Human-Readable Durations]
**Learning:** For long-running processes, displaying uptime in raw hours (e.g., "50h") forces users to do mental math. Breaking it down into days (e.g., "2d 2h") respects the user's cognitive load and aligns with standard CLI patterns.
**Action:** Use a day-aware duration formatter for time spans >= 24 hours.
## 2026-07-05 - Better Empty-State UX for Daemon CLI
**Learning:** For a CLI daemon like sentrylogmon, if required configuration is missing, exiting with a generic one-line error isn't helpful enough. However, immediately checking `len(os.Args) == 1` to print usage is bad because daemons often use default config files. Printing usage *when* validation fails provides the best contextual help.
**Action:** When handling required configuration failures (like missing DSN or log source), call `flag.Usage()` (or equivalent `printUsage()`) right before printing the fatal error to help users discover the required flags and file formats.
