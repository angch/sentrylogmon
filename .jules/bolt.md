## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-06-06 - time.Parse Performance Overhead
**Learning:** In Go, `time.Parse` is surprisingly slow when used in hot loops for log parsing because it handles many complex formats and allocates memory for parsing strings.
**Action:** When the format of the timestamp is strict and known (e.g., Nginx error logs), replace `time.Parse` with manual byte parsing (`atoi4`/`atoi2`) and construct the time directly using `time.Date(..., time.UTC)` for significant performance gains (approx. 4x speedup in this case).
