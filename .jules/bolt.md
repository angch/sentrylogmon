## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-13 - [Fast-path Nginx Timestamp Parsing]
**Learning:** Go's `time.Parse` and regex incur heavy overheads even for exact match string formats, creating an unnecessary performance bottleneck in high-throughput log parsing (e.g., Nginx access/error logs).
**Action:** Use manual byte scanning to extract timestamps (with helpers like `atoi4`/`atoi2`) followed by `time.Date()` which can yield >8x performance improvements and reduce allocations drastically for exact known formats.
