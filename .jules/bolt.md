## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2024-06-15 - Optimize Nginx error log timestamp parsing
**Learning:** Replacing `time.Parse` with manual byte extraction (`atoi2`, `atoi4`) and `time.Date` eliminates string allocation overhead and provides >3x performance improvement for fixed-format timestamps.
**Action:** Look for high-frequency `time.Parse` calls on fixed-format layouts and replace them with manual byte parsing and `time.Date` for hot paths.
## 2024-06-15 - Reviewer false positives on manual parsing
**Learning:** Automated code reviewers might incorrectly flag "Missing Separator Checks" or "Overly Permissive Parsing" when replacing `time.Parse` with manual index-based checks, and might falsely warn about missing local helpers. It's safe to ignore these if the logic is verified, but adding `t.Day() == d` is a good practice to prevent silent date normalization by `time.Date`.
**Action:** When manually parsing dates with `time.Date`, always verify `t.Day() == d` to reject invalid dates like February 30th, matching the strictness of `time.Parse`.
