## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2023-10-27 - Manual Timestamp Parsing for Nginx Error Logs
**Learning:** Replacing `time.Parse` with manual byte parsing for fixed-format timestamps (like Nginx error logs) significantly reduces overhead. It bypasses reflection and layout processing.
**Action:** Always consider manual byte parsing (e.g., using `atoi2`, `atoi4`) for high-throughput, fixed-format timestamp logs where `time.Parse` becomes a significant CPU bottleneck.
