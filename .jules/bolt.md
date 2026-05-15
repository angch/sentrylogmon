## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-02-09 - time.Parse Overhead
**Learning:** `time.Parse` has significant overhead (around ~250ns) even for straightforward timestamp layouts, whereas manual byte extraction to integers and `time.Date` creation is significantly faster (around ~77ns).
**Action:** In log parsing or detector functions that require zero-allocation or high performance, avoid `time.Parse` and string allocation overhead for strictly defined formats. Instead, manually extract date/time components directly from the byte slice and use `time.Date(..., time.UTC)`.
