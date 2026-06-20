## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-03-12 - Replacing time.Parse with manual indexing
**Learning:** `time.Parse` has significant overhead due to format string reflection and substring allocations. Manually indexing characters for known fixed-format timestamps and converting substrings to integers via inline byte operations (e.g. `atoi4`) paired with `time.Date` drops allocations and increases execution speed significantly.
**Action:** Always consider manual parsing for high-throughput logging formats where timestamps are rigidly structured, as avoiding `time.Parse` is an easy win for both CPU and memory.
