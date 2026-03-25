## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-23 - Avoiding stringification allocations with `any`/`interface{}`
**Learning:** In Go hot paths, using `fmt.Sprintf("%v", val)` to extract strings from `any`/`interface{}` values causes memory allocations and is significantly slower (~135ns).
**Action:** Always attempt type assertions like `if s, ok := val.(string); ok` first. This avoids allocations and executes an order of magnitude faster (~0.5ns), falling back to `fmt.Sprintf` only when necessary.
