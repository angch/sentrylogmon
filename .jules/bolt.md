## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-13 - JSON Parsing Fast-Path
**Learning:** Pre-calculating `searchBytes` using `json.Marshal(field)` and using `bytes.Contains` allows skipping expensive `json.Unmarshal` for non-matching lines, drastically improving performance (from ~2656 ns/op to ~90 ns/op, and 19 to 0 allocations). Furthermore, using type assertions like `if s, ok := val.(string); ok` is significantly faster than `fmt.Sprintf("%v", val)` for extracting strings from `interface{}`.
**Action:** Use `bytes.Contains` fast-path for high-throughput line filtering, and prefer type assertions over string formatting for known types.
