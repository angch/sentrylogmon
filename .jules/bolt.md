## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-04-02 - Fast-pathing JSON extraction
**Learning:** Heavy functions like `json.Unmarshal` and `fmt.Sprintf` can significantly slow down the hot path of JSON log parsing. Skipping `json.Unmarshal` via a `bytes.Contains` check of a pre-marshalled field name, and avoiding `fmt.Sprintf` type casting overhead with type assertions, yields significant performance improvements (orders of magnitude faster).
**Action:** In hot loops, always prefer pre-calculating search patterns, early fast-path rejections, and type assertions over reflection-based casting.
