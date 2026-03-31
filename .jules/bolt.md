## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2026-03-31 - Fast-pathing type assertions for string extraction
**Learning:** In Go hot paths, using `fmt.Sprintf("%v", val)` on `any`/`interface{}` values to extract strings causes allocations and is slow. Using type assertions like `if s, ok := val.(string); ok` first has zero allocations and is orders of magnitude faster when the value is indeed a string.
**Action:** Always attempt a type assertion for expected types (like `string`, `int`, `float64`) from `any`/`interface{}` before falling back to reflection-based or formatting functions like `fmt.Sprintf`.
