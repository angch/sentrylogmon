## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2024-07-04 - Optimize ParseNginxError Timestamp Parsing
**Learning:** In Go, `time.Parse` can introduce unnecessary overhead and heap allocations during log parsing. Manual index-based parsing using lightweight helpers (`atoi4`, `atoi2`) avoids allocations and is significantly faster, achieving roughly a 3.5x speedup for Nginx error logs. Note: Default `time.Parse` without timezone parses as UTC, so we explicitly pass `time.UTC` to `time.Date`.
**Action:** Prioritize manual parsing with `time.Date` over `time.Parse` for high-throughput string parsing when the layout is strictly fixed and predictable.
