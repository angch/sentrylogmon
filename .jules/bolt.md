## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Fast-path rejection before heavy parsing
**Learning:** In log processing hot paths, full payload parsing (e.g. JSON) dominates CPU time and allocations. Failing fast on non-matching lines using byte sequence rejection (`bytes.Contains`, `.windows().any()`, `indexOf`) prior to full parsing yields order-of-magnitude (24x-300x) performance improvements.
**Action:** Always apply fast-path byte sequence rejection before complex deserialization to dramatically reduce overhead for non-matching inputs.
