## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - time.Parse Overhead in Hot Paths
**Learning:** `time.Parse` incurs significant overhead, particularly in log parsing where parsing frequency is extremely high. Even with a simple format, it performs allocations and extensive validations that aren't necessary when the format is strict and fixed.
**Action:** For extremely hot paths parsing simple static log timestamps (like Nginx `YYYY/MM/DD HH:MM:SS`), avoid `time.Parse`. Instead, use fixed-index byte slicing and custom `atoi` implementations combined with `time.Date(...)` to completely eliminate allocations and reduce CPU overhead.
