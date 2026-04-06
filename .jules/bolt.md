## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Fast-Path Rejection in JSON parsing
**Learning:** The `JsonDetector` optimization pre-calculates `searchBytes` using `json.Marshal(field)` (avoiding manual colon appending to tolerate spaces like `"key" :` in raw JSON) and uses `bytes.Contains` to skip `json.Unmarshal` for lines missing the target field. This fast-path rejection improves performance significantly (e.g., from ~1671 ns/op down to ~17 ns/op) for non-matching lines.
**Action:** When searching for keys in JSON lines, consider scanning for the marshaled key string using `bytes.Contains` before unmarshaling to provide a fast path and avoid large allocations.
