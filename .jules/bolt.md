## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-04-09 - Avoid fmt.Sprintf on any/interface{} for strings in hot paths
**Learning:** Using `fmt.Sprintf("%v", val)` on an `interface{}` to extract a string causes an allocation and takes ~85ns. If the underlying type is already a string, a type assertion `if s, ok := val.(string); ok` takes < 1ns and avoids the allocation entirely.
**Action:** In high-frequency paths (like `Detect()`), always attempt a type assertion first before falling back to `fmt.Sprintf()`.
