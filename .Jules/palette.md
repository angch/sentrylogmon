## 2026-03-04 - Graceful empty state for Zig CLI status
**Learning:** The CLI status table lacked a friendly empty state when there were no instances, printing only the table header, which can be confusing for a user. In Go and Rust, 'No running instances found.' is printed instead.
**Action:** Added a check for empty instances and printed a helpful user-facing message to provide a better UX in the Zig CLI implementation.
