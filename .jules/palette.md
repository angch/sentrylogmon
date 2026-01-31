## 2026-01-30 - CLI Output Humanization
**Learning:** CLI tools often default to JSON for machine readability, alienating human users. TTY detection (`os.ModeCharDevice`) allows serving both audiences perfectly: tables for humans, JSON for pipes/scripts.
**Action:** Always check `isTerminal` before outputting status or list data in CLI tools; default to pretty tables for humans.

## 2026-01-31 - Truncation with Detail
**Learning:** When summarizing complex lists (like active monitors) in a limited space (table cells), context is king. A format like `name(type)` is far more valuable to a user than a raw count (e.g., "3 monitors"), even if it requires truncation. Users can infer the rest, but a count gives them nothing.
**Action:** Prioritize dense, identifying information in status tables over aggregate counts. Use safe rune-based truncation to prevent text corruption.
