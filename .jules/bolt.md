## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-24 - JSON Parsing Optimization with Fast-Path Rejection
**Learning:** `json.Unmarshal` is extremely slow and allocation-heavy when processing high-throughput logs where the target field may not even be present in the line. By pre-calculating the target field's JSON representation using `json.Marshal(field)` and using `bytes.Contains` as a fast-path rejection, we can skip unmarshaling entirely for non-matching lines, resulting in a ~99% performance improvement (from ~1671 ns/op to ~17 ns/op). Furthermore, type asserting to `string` before falling back to `fmt.Sprintf("%v", val)` eliminates allocations and improves performance for strings (the most common JSON value type).
**Action:** Always consider fast-path string/byte scanning rejections before invoking heavy parsers like `json.Unmarshal` or `regexp`, and prioritize type assertions over reflection-based string formatting (`fmt.Sprintf`) in hot paths.
