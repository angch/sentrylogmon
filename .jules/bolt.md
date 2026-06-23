## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2025-02-27 - Replace time.Parse with manual parsing for Nginx Error timestamps
**Learning:** In Go, replacing `time.Parse` with manual byte extraction and `time.Date` for fixed-format timestamps (like Nginx error logs `2023/10/27 10:00:00`) avoids string allocation overhead and reflection, leading to significant performance improvements. Manual parsing drops from ~245ns/op to ~70ns/op, a >3x speedup.
**Action:** When extracting timestamps from high-throughput log lines with fixed, known formats, use custom `atoi` byte-parsing helpers and `time.Date` instead of `time.Parse` if benchmarking confirms a bottleneck or notable gain.
