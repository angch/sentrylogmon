## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-15 - Fast-Path Rejection for Expensive Operations
**Learning:** `json.Unmarshal` is extremely expensive, taking ~2200ns and causing multiple allocations even when the log line ultimately doesn't match the target pattern or field.
**Action:** When searching for specific fields in structured logs, always use a fast-path rejection (e.g., `bytes.Contains` with pre-calculated `json.Marshal(field)`) before attempting the expensive parse. This reduced non-matching line processing from ~2200ns/19 allocs to ~80ns/0 allocs.
