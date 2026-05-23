## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-05-23 - Manual parsing overhead vs reflection
**Learning:** Using `time.Parse` with explicit format strings relies on string allocation and reflection-like behavior, which introduces noticeable overhead in hot paths (parsing logs). Manually parsing byte slices using basic index checking (e.g. `atoi4`, `atoi2`) and passing them directly into `time.Date(..., time.UTC)` results in up to a ~75% reduction in latency (~240ns vs ~60ns).
**Action:** When extracting time data in a performance-critical loop, implement manual extraction and validation, ensuring you rigorously validate the parsed date component bounds to avoid correctness regressions.
