## 2026-01-30 - CLI Output Humanization
**Learning:** CLI tools often default to JSON for machine readability, alienating human users. TTY detection (`os.ModeCharDevice`) allows serving both audiences perfectly: tables for humans, JSON for pipes/scripts.
**Action:** Always check `isTerminal` before outputting status or list data in CLI tools; default to pretty tables for humans.
