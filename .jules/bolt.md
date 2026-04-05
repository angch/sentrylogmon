## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-04-05 - Optimize string conversion of any types
**Learning:** Using `fmt.Sprintf("%v", val)` to cast `any` to a string is highly inefficient (causes allocations and takes ~90ns/op). For hot paths processing logs, when the type is mostly known, utilizing a type assertion first (`if s, ok := val.(string); ok`) avoids allocations and is orders of magnitude faster (~0.39ns/op).
**Action:** Always prefer type assertions over `fmt.Sprintf` for extracting primitive types from `interface{}`/`any` in hot paths. Use `fmt.Sprintf` only as a fallback.
