## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - time.Parse Overhead in Timestamp Extraction
**Learning:** Using `time.Parse` inside high-frequency log parsing paths (like Nginx error logs) introduces significant overhead (around 250ns/op). Replacing it with manual byte parsing (e.g., `atoi4`, `atoi2`) and direct `time.Date` construction drastically reduces the execution time to around 72ns/op, a >3x speedup.
**Action:** For hot-path log detectors with fixed or predictable timestamp formats, always prefer manual byte extraction over standard library parsing functions to minimize CPU cycles and allocations.
