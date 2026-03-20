## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-20 - Manual Byte Parsing Optimization for Log Timestamps
**Learning:** `time.Parse` has significant overhead and allocates memory even when the format is exact. When processing high-throughput log streams where timestamp formats are strict (e.g., Nginx error logs with fixed `YYYY/MM/DD HH:MM:SS` format), manual byte extraction combined with `time.Date()` can yield >8x performance improvements and zero allocations.
**Action:** When optimizing hot paths for date/time parsing where the format is rigid and known, use `atoi` helpers to parse individual fields directly from byte slices and use `time.Date()` instead of relying on `time.Parse`. Ensure basic bounds checking is applied before calling `time.Date()` to avoid silently wrapping invalid dates.
