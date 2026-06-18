## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2024-06-18 - Optimized Nginx Error Timestamp Parsing
**Learning:** Replaced reflection-heavy `time.Parse` with manual byte iteration parsing to avoid expensive string allocations when processing dense Nginx error logs. Found that `time.Date` is drastically faster when components are manually extracted using `atoi2`/`atoi4` local helpers.
**Action:** Always favor manual byte slice slicing over string conversions and generic `time.Parse` for high-throughput logging critical paths.
