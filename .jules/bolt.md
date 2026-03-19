## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-19 - time.Parse in log parsing
**Learning:** For high-throughput log parsing (e.g., Nginx error logs), using `time.Parse` incurs overhead even for exact format matches due to its flexibility.
**Action:** Replace `time.Parse` with manual byte scanning and string extraction, using simple helpers like `atoi4`/`atoi2` and `time.Date()`. This yielded a massive 3.5x performance improvement (250ns -> 70ns) while preserving the same logic.
